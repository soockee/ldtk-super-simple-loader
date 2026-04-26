package ldtkgo

import (
	"encoding/json"
	"image"
	"io"
	"io/fs"
	"path"
)

// TileRect represents a rectangular region within a tileset.
type TileRect struct {
	TilesetUID int `json:"tilesetUid"`
	X          int `json:"x"`
	Y          int `json:"y"`
	W          int `json:"w"`
	H          int `json:"h"`
}

// TilesetDef represents a tileset definition from the .ldtk project file.
type TilesetDef struct {
	Identifier   string      `json:"identifier"`
	UID          int         `json:"uid"`
	RelPath      string      `json:"relPath"`
	PxWid        int         `json:"pxWid"`
	PxHei        int         `json:"pxHei"`
	TileGridSize int         `json:"tileGridSize"`
	Spacing      int         `json:"spacing"`
	Padding      int         `json:"padding"`
	Image        image.Image `json:"-"` // Loaded tileset image (set via LoadTilesetImage)
}

// EntityDef represents an entity definition from the .ldtk project file.
type EntityDef struct {
	Identifier     string    `json:"identifier"`
	UID            int       `json:"uid"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	Color          string    `json:"color"`
	Tags           []string  `json:"tags"`
	RenderMode     string    `json:"renderMode"`
	TilesetID      *int      `json:"tilesetId"`
	TileRect       *TileRect `json:"tileRect"`
	TileRenderMode string    `json:"tileRenderMode"`
}

// LayerDef represents a layer definition from the .ldtk project file.
type LayerDef struct {
	Identifier    string         `json:"identifier"`
	Type          string         `json:"type"`
	UID           int            `json:"uid"`
	GridSize      int            `json:"gridSize"`
	TilesetDefUID *int           `json:"tilesetDefUid"`
	IntGridValues []IntGridValue `json:"intGridValues"`
}

// IntGridValue represents a single value definition in an IntGrid layer.
type IntGridValue struct {
	Value      int    `json:"value"`
	Identifier string `json:"identifier"`
	Color      string `json:"color"`
}

// LevelRef holds minimal level information extracted from the .ldtk project file.
type LevelRef struct {
	Identifier string  `json:"identifier"`
	IID        string  `json:"iid"`
	BGRelPath  *string `json:"bgRelPath"`
	BGPos      *string `json:"bgPos"`
}

// Project represents the relevant definitions from an .ldtk project file.
// Only the definitions needed for the simplified export workflow are extracted.
type Project struct {
	Tilesets   []*TilesetDef
	EntityDefs []*EntityDef
	LayerDefs  []*LayerDef
	LevelRefs  []*LevelRef

	tilesetsByUID   map[int]*TilesetDef
	entityDefsByID  map[string]*EntityDef
	entityDefsByUID map[int]*EntityDef
	levelRefsByID   map[string]*LevelRef
}

type projectJSON struct {
	Defs   defsJSON    `json:"defs"`
	Levels []*LevelRef `json:"levels"`
}

type defsJSON struct {
	Tilesets []*TilesetDef `json:"tilesets"`
	Entities []*EntityDef  `json:"entities"`
	Layers   []*LayerDef   `json:"layers"`
}

// OpenProject loads an .ldtk project file from the file system.
func OpenProject(filepath string, fileSystem fs.FS) (*Project, error) {
	file, err := fileSystem.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return ReadProject(data)
}

// ReadProject parses an .ldtk project from raw bytes.
// Only definitions (tilesets, entities, layers) and level references are extracted.
func ReadProject(data []byte) (*Project, error) {
	var raw projectJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	p := &Project{
		Tilesets:        raw.Defs.Tilesets,
		EntityDefs:      raw.Defs.Entities,
		LayerDefs:       raw.Defs.Layers,
		LevelRefs:       raw.Levels,
		tilesetsByUID:   make(map[int]*TilesetDef, len(raw.Defs.Tilesets)),
		entityDefsByID:  make(map[string]*EntityDef, len(raw.Defs.Entities)),
		entityDefsByUID: make(map[int]*EntityDef, len(raw.Defs.Entities)),
		levelRefsByID:   make(map[string]*LevelRef, len(raw.Levels)),
	}

	for _, t := range p.Tilesets {
		p.tilesetsByUID[t.UID] = t
	}
	for _, e := range p.EntityDefs {
		p.entityDefsByID[e.Identifier] = e
		p.entityDefsByUID[e.UID] = e
	}
	for _, l := range p.LevelRefs {
		p.levelRefsByID[l.Identifier] = l
	}

	return p, nil
}

// TilesetByUID returns the tileset definition with the given UID, or nil.
func (p *Project) TilesetByUID(uid int) *TilesetDef {
	return p.tilesetsByUID[uid]
}

// EntityDefByID returns the entity definition with the given identifier, or nil.
func (p *Project) EntityDefByID(id string) *EntityDef {
	return p.entityDefsByID[id]
}

// LevelRefByID returns the level reference with the given identifier, or nil.
func (p *Project) LevelRefByID(id string) *LevelRef {
	return p.levelRefsByID[id]
}

// LoadTilesetImage loads a single tileset's image from the file system.
// basePath is the directory containing the .ldtk file.
func (p *Project) LoadTilesetImage(tileset *TilesetDef, basePath string, fileSystem fs.FS) error {
	filePath := path.Join(basePath, tileset.RelPath)
	img, err := loadPNG(filePath, fileSystem)
	if err != nil {
		return err
	}
	tileset.Image = img
	return nil
}

// LoadAllTilesetImages loads all tileset images from the file system.
// basePath is the directory containing the .ldtk file.
func (p *Project) LoadAllTilesetImages(basePath string, fileSystem fs.FS) error {
	for _, t := range p.Tilesets {
		if err := p.LoadTilesetImage(t, basePath, fileSystem); err != nil {
			return err
		}
	}
	return nil
}
