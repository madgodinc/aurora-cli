package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

func NotebookEditTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "NotebookEdit",
			Description: "Edit a Jupyter notebook cell. Specify cell index (0-based) and new source.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path":  map[string]interface{}{"type": "string"},
					"cell_index": map[string]interface{}{"type": "integer"},
					"new_source": map[string]interface{}{"type": "string"},
					"cell_type":  map[string]interface{}{"type": "string", "enum": []string{"code", "markdown"}},
				},
				"required": []string{"file_path", "cell_index", "new_source"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			fp, _ := input["file_path"].(string)
			idx := int(input["cell_index"].(float64))
			src, _ := input["new_source"].(string)
			cellType, _ := input["cell_type"].(string)

			data, err := os.ReadFile(fp)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			var nb map[string]interface{}
			if err := json.Unmarshal(data, &nb); err != nil {
				return fmt.Sprintf("Parse error: %v", err)
			}
			cells, ok := nb["cells"].([]interface{})
			if !ok || idx < 0 || idx >= len(cells) {
				return fmt.Sprintf("Cell index %d out of range (0-%d)", idx, len(cells)-1)
			}
			cell, _ := cells[idx].(map[string]interface{})
			cell["source"] = strings.Split(src, "\n")
			cell["outputs"] = []interface{}{}
			if cellType != "" {
				cell["cell_type"] = cellType
			}
			out, _ := json.MarshalIndent(nb, "", " ")
			os.WriteFile(fp, out, 0644)
			return fmt.Sprintf("Cell %d updated in %s", idx, fp)
		},
	}
}
