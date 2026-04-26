package ldtkgo

import (
	"encoding/csv"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

// IntGrid represents a loaded IntGrid layer as a 2D grid of integer values.
// Rows are indexed by Y (top to bottom), columns by X (left to right).
type IntGrid struct {
	// Name is the layer identifier from the project (e.g., "collision").
	Name string

	// Def is the layer definition from the project, if linked.
	Def *LayerDef

	// Grid holds the cell values. Grid[row][col] where row=y, col=x.
	Grid [][]int

	// Width is the number of columns.
	Width int

	// Height is the number of rows.
	Height int
}

// At returns the IntGrid value at the given cell coordinates.
// Returns 0 for out-of-bounds coordinates.
func (ig *IntGrid) At(col, row int) int {
	if row < 0 || row >= ig.Height || col < 0 || col >= ig.Width {
		return 0
	}
	return ig.Grid[row][col]
}

// AtPx returns the IntGrid value at the given pixel coordinates.
// Requires a linked Def with GridSize > 0. Returns 0 if unlinked or out of bounds.
func (ig *IntGrid) AtPx(px, py int) int {
	if ig.Def == nil || ig.Def.GridSize <= 0 {
		return 0
	}
	return ig.At(px/ig.Def.GridSize, py/ig.Def.GridSize)
}

// loadIntGridCSV loads a single IntGrid CSV file and returns an IntGrid.
// The CSV name should match the layer identifier (e.g., "collision_test.csv").
func loadIntGridCSV(name string, filePath string, fileSystem fs.FS) (*IntGrid, error) {
	file, err := fileSystem.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseIntGridCSV(name, file)
}

// parseIntGridCSV parses an IntGrid from a CSV reader.
func parseIntGridCSV(name string, r io.Reader) (*IntGrid, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // allow variable (trailing comma)

	ig := &IntGrid{Name: name}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var row []int
		for _, field := range record {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			val, err := strconv.Atoi(field)
			if err != nil {
				return nil, err
			}
			row = append(row, val)
		}
		if len(row) > 0 {
			ig.Grid = append(ig.Grid, row)
		}
	}

	ig.Height = len(ig.Grid)
	if ig.Height > 0 {
		ig.Width = len(ig.Grid[0])
	}

	return ig, nil
}

// LoadIntGrids discovers and loads all IntGrid CSV files from a simplified export directory.
// CSV files are matched against IntGrid layer definitions from the project.
// dir is the path to the level folder (e.g., "pong/simplified/level_0").
func (level *Level) LoadIntGrids(dir string, fileSystem fs.FS, project *Project) error {
	if level.IntGrids == nil {
		level.IntGrids = map[string]*IntGrid{}
	}

	for _, layerDef := range project.LayerDefs {
		if layerDef.Type != "IntGrid" {
			continue
		}
		csvPath := path.Join(dir, layerDef.Identifier+".csv")
		ig, err := loadIntGridCSV(layerDef.Identifier, csvPath, fileSystem)
		if err != nil {
			// CSV might not exist if layer is empty — skip silently
			continue
		}
		ig.Def = layerDef
		level.IntGrids[layerDef.Identifier] = ig
	}

	return nil
}
