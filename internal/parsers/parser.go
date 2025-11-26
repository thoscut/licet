package parsers

import (
	"fmt"

	"github.com/thoscut/licet/internal/models"
)

// Parser interface for all license server types
type Parser interface {
	Query(hostname string) models.ServerQueryResult
}

// ParserFactory creates appropriate parser for license server type
type ParserFactory struct {
	lmutilPath     string
	rlmstatPath    string
	spmstatPath    string
	sesictrlPath   string
	rvlstatusPath  string
	tlmServerPath  string
	pixarQueryPath string
}

func NewParserFactory(binPaths map[string]string) *ParserFactory {
	return &ParserFactory{
		lmutilPath:     binPaths["lmutil"],
		rlmstatPath:    binPaths["rlmstat"],
		spmstatPath:    binPaths["spmstat"],
		sesictrlPath:   binPaths["sesictrl"],
		rvlstatusPath:  binPaths["rvlstatus"],
		tlmServerPath:  binPaths["tlm_server"],
		pixarQueryPath: binPaths["pixar_query"],
	}
}

func (f *ParserFactory) GetParser(serverType string) (Parser, error) {
	switch serverType {
	case "flexlm":
		return NewFlexLMParser(f.lmutilPath), nil
	case "rlm":
		return NewRLMParser(f.rlmstatPath), nil
	// TODO: Implement other parsers
	// case "spm":
	//     return NewSPMParser(f.spmstatPath), nil
	// case "sesi":
	//     return NewSESIParser(f.sesictrlPath), nil
	// case "rvl":
	//     return NewRVLParser(f.rvlstatusPath), nil
	// case "tweak":
	//     return NewTweakParser(f.tlmServerPath), nil
	// case "pixar":
	//     return NewPixarParser(f.pixarQueryPath), nil
	default:
		return nil, fmt.Errorf("unsupported server type: %s", serverType)
	}
}
