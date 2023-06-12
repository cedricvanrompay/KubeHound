package graphdb

import (
	"context"
	"errors"
	"sync"

	"github.com/DataDog/KubeHound/pkg/kubehound/graph/vertex"
	"github.com/DataDog/KubeHound/pkg/telemetry/log"
	gremlingo "github.com/apache/tinkerpop/gremlin-go/driver"
)

var _ AsyncVertexWriter = (*JanusGraphAsyncVertexWriter)(nil)

type JanusGraphAsyncVertexWriter struct {
	gremlin         vertex.VertexTraversal
	traversalSource *gremlingo.GraphTraversalSource
	inserts         []vertex.TraversalInput
	consumerChan    chan []vertex.TraversalInput
	writingInFlight *sync.WaitGroup
	batchSize       int
	mu              sync.Mutex
}

func NewJanusGraphAsyncVertexWriter(ctx context.Context, drc *gremlingo.DriverRemoteConnection, v vertex.Builder, opts ...WriterOption) (*JanusGraphAsyncVertexWriter, error) {
	options := &writerOptions{}
	for _, opt := range opts {
		opt(options)
	}

	source := gremlingo.Traversal_().WithRemote(drc)
	jw := JanusGraphAsyncVertexWriter{
		gremlin:         v.Traversal(),
		inserts:         make([]vertex.TraversalInput, 0, v.BatchSize()),
		traversalSource: source,
		batchSize:       v.BatchSize(),
		writingInFlight: &sync.WaitGroup{},
		consumerChan:    make(chan []vertex.TraversalInput, v.BatchSize()*channelSizeBatchFactor),
	}
	jw.startBackgroundWriter(ctx)
	return &jw, nil
}

// startBackgroundWriter starts a background go routine
func (jgv *JanusGraphAsyncVertexWriter) startBackgroundWriter(ctx context.Context) {
	go func() {
		for {
			select {
			case data := <-jgv.consumerChan:
				// closing the channel shoud stop the go routine
				if data == nil {
					return
				}
				err := jgv.batchWrite(ctx, data)
				if err != nil {
					log.I.Errorf("write data in background batch writer: %v", err)
				}
			case <-ctx.Done():
				log.I.Info("Closed background janusgraph worker")
				return
			}
		}
	}()
}

func (jgv *JanusGraphAsyncVertexWriter) batchWrite(ctx context.Context, data []vertex.TraversalInput) error {
	log.I.Debugf("batch write JanusGraphAsyncVertexWriter with %d elements", len(data))
	defer jgv.writingInFlight.Done()

	op := jgv.gremlin(jgv.traversalSource, data)
	promise := op.Iterate()
	err := <-promise
	if err != nil {
		return err
	}
	return nil
}

func (jgv *JanusGraphAsyncVertexWriter) Close(ctx context.Context) error {
	close(jgv.consumerChan)
	return nil
}

// Flush triggers writes of any remaining items in the queue.
// This is blocking
func (jgv *JanusGraphAsyncVertexWriter) Flush(ctx context.Context) error {
	jgv.mu.Lock()
	defer jgv.mu.Unlock()

	if jgv.traversalSource == nil {
		return errors.New("JanusGraph traversalSource is not initialized")
	}

	if len(jgv.inserts) != 0 {
		jgv.writingInFlight.Add(1)
		err := jgv.batchWrite(ctx, jgv.inserts)
		if err != nil {
			log.I.Errorf("batch write vertex: %+v", err)
			jgv.writingInFlight.Wait()
			return err
		}
		log.I.Info("Done flushing vertices, clearing the queue")
		jgv.inserts = nil
	}

	jgv.writingInFlight.Wait()

	return nil
}

func (jgv *JanusGraphAsyncVertexWriter) Queue(ctx context.Context, v any) error {
	jgv.mu.Lock()
	defer jgv.mu.Unlock()

	jgv.inserts = append(jgv.inserts, v)
	if len(jgv.inserts) > jgv.batchSize {
		copied := make([]vertex.TraversalInput, len(jgv.inserts))
		copy(copied, jgv.inserts)
		jgv.writingInFlight.Add(1)
		jgv.consumerChan <- copied
		// cleanup the ops array after we have copied it to the channel
		jgv.inserts = nil
	}
	return nil
}
