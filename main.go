package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

const (
	jsonFormat = "json"
	hclFormat  = "hcl"
)

func main() {
	// Command-line flags
	inputFile := flag.String("i", "", "Input file path")
	outputFile := flag.String("o", "", "Output file path")
	reverse := flag.Bool("r", false, "Reverse input/output format")
	version := flag.Bool("v", false, "Show version")
	help := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *version {
		fmt.Println("json2hcl2 version 1.0")
		return
	}

	if *help {
		flag.Usage()
		return
	}

	inputData, err := readInputData(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read input: %s", err)
	}

	outputData, err := convertData(inputData, *reverse)
	if err != nil {
		log.Fatalf("Failed to convert data: %s", err)
	}

	if *outputFile == "" {
		writeOutputData(os.Stdout, outputData)
	} else {
		err = writeOutputFile(*outputFile, outputData)
		if err != nil {
			log.Fatalf("Failed to write output file: %s", err)
		}
	}
}

func readInputData(inputFile string) ([]byte, error) {
	if inputFile == "" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(inputFile)
}

func convertData(inputData []byte, reverse bool) ([]byte, error) {
	if reverse {
		return convertToJSON(inputData)
	}
	return convertToHCL(inputData)
}

func parse(reverse bool, inputData []byte, filename string) (*hcl.File, error) {
	parser := hclparse.NewParser()
	if reverse {
		parsedFile, diag := parser.ParseJSON(inputData, filename)
		if diag != nil {
			return nil, fmt.Errorf("failed to parse JSON: %s", diag.Error())
		}
		return parsedFile, diag
	} else {
		parsedFile, diag := parser.ParseHCL(inputData, filename)
		if diag != nil {
			return nil, fmt.Errorf("failed to parse HCL: %s", diag.Error())
		}
		return parsedFile, diag
	}
}

func convertToJSON(inputData []byte) ([]byte, error) {
	hclFile, _ := parse(false, inputData, hclFormat)
	if _, diag := hclFile.Body.JustAttributes(); !diag.HasErrors() {
		file := hclwrite.NewEmptyFile()
		return hclToJSON(file)
	} else {
		return nil, fmt.Errorf("failed to parse JSON: %s", diag.Error())
	}
}

func convertToHCL(inputData []byte) ([]byte, error) {
	var obj map[string]interface{}
	hclFile, _ := parse(true, inputData, jsonFormat)
	if _, diag := hclFile.Body.JustAttributes(); !diag.HasErrors() {
		err := json.Unmarshal(inputData, &obj)
		if err != nil {
			return nil, fmt.Errorf("failed to Unmarshal JSON: %s", err)
		}
		file := hclwrite.NewEmptyFile()
		jsonToHCL(file.Body(), obj)
		return file.Bytes(), nil
	} else {
		return nil, fmt.Errorf("failed to parse JSON: %s", diag.Error())
	}
}

func writeOutputData(outputStream *os.File, outputData []byte) {
	outputStream.Write(outputData)
}

func writeOutputFile(outputFile string, outputData []byte) error {
	return ioutil.WriteFile(outputFile, outputData, 0644)
}

func hclToJSON(file *hclwrite.File) ([]byte, error) {
	return json.MarshalIndent(file.Body().BuildTokens(nil), "", "  ")
}

func jsonToHCL(body *hclwrite.Body, obj map[string]interface{}) {
	for key, value := range obj {
		switch value.(type) {
		case string, bool, float64, int64:
			body.SetAttributeValue(key, ctyFromValue(value))
		case []string, []float64, []int64:
			body.SetAttributeValue(key, ctyFromValue(value))
		case []interface{}:
			body.SetAttributeValue(key, ctyFromValue(value))
		case map[string]interface{}:
			body.SetAttributeValue(key, ctyFromValue(value))
		}
	}
}

func ctyFromJson(value interface{}) cty.Value {
	jsonVal, err := json.Marshal(value)
	if err != nil {
		log.Fatalf("failed to marshal json value: %s", err.Error())
		return cty.NilVal
	}
	ctyType, err := ctyjson.ImpliedType(jsonVal)
	if err != nil {
		log.Fatalf("failed to get ctyType: %s", err.Error())
		return cty.NilVal
	}
	ctyVal, err := ctyjson.Unmarshal(jsonVal, ctyType)
	if err != nil {
		log.Fatalf("failed to get ctyVal: %s", err.Error())
		return cty.NilVal
	}
	return ctyVal
}

func ctyFromValue(value interface{}) cty.Value {
	switch v := value.(type) {
	case string:
		return cty.StringVal(v)
	case bool:
		return cty.BoolVal(v)
	case float64:
		return cty.NumberFloatVal(v)
	case int64:
		return cty.NumberIntVal(v)
	case []interface{}:
		return ctyFromJson(value)
	case []string, []float64, []int64:
		return ctyFromJson(value)
	case map[string]interface{}:
		return ctyFromJson(value)
	}
	return cty.NilVal
}
