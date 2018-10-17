package json

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/caalberts/localroast/types"
	log "github.com/sirupsen/logrus"
	"regexp"
)

type Parser struct {
	output chan []types.Schema
}

func NewParser() *Parser {
	return &Parser{
		output: make(chan []types.Schema),
	}
}

func (p *Parser) Output() chan []types.Schema {
	return p.output
}

func (p *Parser) Watch(input chan io.Reader) {
	go func() {
		for input := range input {
			var stubs []stub
			err := json.NewDecoder(input).Decode(&stubs)
			if err != nil {
				log.Error(err)
			} else {
				schemas, err := createSchema(stubs)
				if err != nil {
					log.Error(err)
				} else {
					log.Info("parsed new schema")
					p.output <- schemas
				}
			}
		}
	}()
}

func createSchema(stubs []stub) ([]types.Schema, error) {
	schemas := make([]types.Schema, len(stubs))

	for i := 0; i < len(stubs); i++ {
		if f := missingFields(stubs[i]); len(f) > 0 {
			err := fmt.Errorf("missing required fields: %s", strings.Join(f, ", "))
			return []types.Schema{}, err
		}
		schemas[i] = types.Schema{
			Method:   *stubs[i].Method,
			Path:     *stubs[i].Path,
			Status:   *stubs[i].Status,
			Response: formatResponse(stubs[i].Response),
		}
	}

	return schemas, nil
}

type stub struct {
	Method   *string         `json:"method"`
	Path     *string         `json:"path"`
	Status   *int            `json:"status"`
	Response json.RawMessage `json:"response"`
}

func formatResponse(rawJSON json.RawMessage) []byte {
	repeatedWhitespace := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	return repeatedWhitespace.ReplaceAll([]byte(rawJSON), []byte(" "))
}

func missingFields(s stub) []string {
	var missingFields []string
	if s.Method == nil {
		missingFields = append(missingFields, "method")
	}
	if s.Path == nil {
		missingFields = append(missingFields, "path")
	}
	if s.Status == nil {
		missingFields = append(missingFields, "status")
	}
	return missingFields
}
