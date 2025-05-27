package urlprocessor

import (
	"net/url"
)

// a centralized way to process URLs.
type Registry struct {
	processors       []Processor
	defaultProcessor Processor
}

// NewRegistry creates a new processor registry with default processors.
func NewRegistry() *Registry {
	return &Registry{
		processors: []Processor{
			NewGitHubProcessor(nil),
			// Add more processors here as needed
		},
		defaultProcessor: &DefaultProcessor{},
	}
}

// AddProcessor adds a new processor to the registry.
func (r *Registry) AddProcessor(processor Processor) {
	r.processors = append(r.processors, processor)
}

// ProcessURL processes a URL with the appropriate processor.
func (r *Registry) ProcessURL(sourceURL *url.URL, raw bool) string {
	for _, processor := range r.processors {
		if processor.CanProcess(sourceURL) {
			return processor.Process(sourceURL, raw)
		}
	}

	// Fallback to default processor if no other processor can handle it
	return r.defaultProcessor.Process(sourceURL, raw)
}

// ProcessURLString processes a URL string with the appropriate processor.
func (r *Registry) ProcessURLString(urlStr string, raw bool) (string, error) {
	sourceURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	return r.ProcessURL(sourceURL, raw), nil
}
