// Package ldtkgo is a loader for LDtk "Super Simple Export" data.json files, written in Go.
// The super simple export produces per-level folders containing a data.json with level metadata,
// entity instances, and custom fields, along with pre-rendered PNG layers and composites.
//
// Use Open() to load from a file system, or Read() to load from raw bytes.
package ldtkgo

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"io"
	"io/fs"
)

// NeighbourLevel represents a neighbouring level reference in the simplified export.
type NeighbourLevel struct {
	LevelIID string `json:"levelIid"`
	Dir      string `json:"dir"` // Direction: "n", "s", "e", "w", "ne", "nw", "se", "sw", "o", ">", "<"
}

// Entity represents an entity instance in the simplified export.
type Entity struct {
	ID           string                 `json:"id"`           // Entity definition identifier (name)
	IID          string                 `json:"iid"`          // Unique instance identifier
	Layer        string                 `json:"layer"`        // Layer identifier the entity belongs to
	X            int                    `json:"x"`            // X position in pixels
	Y            int                    `json:"y"`            // Y position in pixels
	Width        int                    `json:"width"`        // Width in pixels
	Height       int                    `json:"height"`       // Height in pixels
	Color        int                    `json:"color"`        // Smart color as integer
	CustomFields map[string]interface{} `json:"customFields"` // Custom field values
	Def          *EntityDef             `json:"-"`            // Entity definition (set via LinkProject)
	Tileset      *TilesetDef            `json:"-"`            // Resolved tileset (set via LinkProject)
	Tags         []string               `json:"-"`            // Tags inherited from EntityDef (set via LinkProject)
	Data         interface{}            `json:"-"`            // User-attached data (not from JSON)
}

// ColorRGBA returns the entity's smart color as a color.RGBA value.
func (e *Entity) ColorRGBA() color.RGBA {
	return color.RGBA{
		R: uint8((e.Color >> 16) & 0xFF),
		G: uint8((e.Color >> 8) & 0xFF),
		B: uint8(e.Color & 0xFF),
		A: 0xFF,
	}
}

// Pos returns the entity's position as (x, y).
func (e *Entity) Pos() (int, int) {
	return e.X, e.Y
}

// Size returns the entity's dimensions as (width, height).
func (e *Entity) Size() (int, int) {
	return e.Width, e.Height
}

// Rect returns the entity's bounding rectangle.
func (e *Entity) Rect() image.Rectangle {
	return image.Rect(e.X, e.Y, e.X+e.Width, e.Y+e.Height)
}

// subImager is implemented by most image types from the standard library.
type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

// SubImage returns the tile rect sub-image from the entity's resolved tileset.
// Returns nil if the entity has no linked definition, tile rect, or loaded tileset image.
func (e *Entity) SubImage() image.Image {
	if e.Def == nil || e.Def.TileRect == nil || e.Tileset == nil || e.Tileset.Image == nil {
		return nil
	}
	si, ok := e.Tileset.Image.(subImager)
	if !ok {
		return nil
	}
	tr := e.Def.TileRect
	return si.SubImage(image.Rect(tr.X, tr.Y, tr.X+tr.W, tr.Y+tr.H))
}

// Level represents a single level from the simplified export's data.json.
type Level struct {
	Identifier      string                 `json:"identifier"`      // Level name
	UniqueIdentifer string                 `json:"uniqueIdentifer"` // Level IID (note: LDtk typo preserved)
	X               int                    `json:"x"`               // World X position
	Y               int                    `json:"y"`               // World Y position
	Width           int                    `json:"width"`           // Width in pixels
	Height          int                    `json:"height"`          // Height in pixels
	BGColorString   string                 `json:"bgColor"`         // Background color as hex string
	BGColor         color.Color            `json:"-"`               // Parsed background color
	BGRelPath       string                 `json:"-"`               // Background image path (set via LinkProject)
	BGImage         image.Image            `json:"-"`               // Loaded background image
	NeighbourLevels []*NeighbourLevel      `json:"neighbourLevels"` // Adjacent levels
	CustomFields    map[string]interface{} `json:"customFields"`    // Level custom field values
	Layers          []string               `json:"layers"`          // Layer PNG filenames
	Entities        map[string][]*Entity   `json:"entities"`        // Entities grouped by identifier
	LoadedLayers    []*Layer               `json:"-"`               // Loaded layer images (set via LoadLayers or LoadWorld)
	IntGrids        map[string]*IntGrid    `json:"-"`               // Loaded IntGrid layers (set via LoadIntGrids or LoadWorld)
	Data            interface{}            `json:"-"`               // User-attached data (not from JSON)
}

// EntityByIID returns the Entity with the given unique identifier, or nil if not found.
func (level *Level) EntityByIID(iid string) *Entity {
	for _, entities := range level.Entities {
		for _, entity := range entities {
			if entity.IID == iid {
				return entity
			}
		}
	}
	return nil
}

// EntitiesByID returns all Entity instances with the given definition identifier, or nil if not found.
func (level *Level) EntitiesByID(id string) []*Entity {
	if entities, ok := level.Entities[id]; ok {
		return entities
	}
	return nil
}

// Entity returns the first Entity with the given definition identifier, or nil.
// Shorthand for EntitiesByID(id)[0] when you know there is exactly one.
func (level *Level) Entity(id string) *Entity {
	if entities, ok := level.Entities[id]; ok && len(entities) > 0 {
		return entities[0]
	}
	return nil
}

// AllEntities returns a flat slice of all entity instances across all groups.
func (level *Level) AllEntities() []*Entity {
	var all []*Entity
	for _, entities := range level.Entities {
		all = append(all, entities...)
	}
	return all
}

// EntitiesByTag returns all entity instances that have the given tag.
// Tags are inherited from EntityDef during LinkProject.
func (level *Level) EntitiesByTag(tag string) []*Entity {
	var result []*Entity
	for _, entities := range level.Entities {
		for _, entity := range entities {
			for _, t := range entity.Tags {
				if t == tag {
					result = append(result, entity)
					break
				}
			}
		}
	}
	return result
}

// IntGrid returns the IntGrid with the given layer identifier, or nil if not found.
func (level *Level) IntGrid(name string) *IntGrid {
	if level.IntGrids == nil {
		return nil
	}
	return level.IntGrids[name]
}

// LinkProject links entity instances with their definitions and tilesets from the project.
// Also sets BGRelPath from the project's level reference.
func (level *Level) LinkProject(project *Project) {
	// Link background path from project
	if ref := project.LevelRefByID(level.Identifier); ref != nil && ref.BGRelPath != nil {
		level.BGRelPath = *ref.BGRelPath
	}

	// Link entity definitions, tilesets, and tags
	for _, entities := range level.Entities {
		for _, entity := range entities {
			if def := project.EntityDefByID(entity.ID); def != nil {
				entity.Def = def
				entity.Tags = def.Tags
				if def.TilesetID != nil {
					entity.Tileset = project.TilesetByUID(*def.TilesetID)
				}
			}
		}
	}
}

// Open loads a simplified data.json from the filepath specified using the file system provided.
func Open(filepath string, fileSystem fs.FS) (*Level, error) {
	file, err := fileSystem.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return Read(bytes)
}

// Read parses a simplified data.json from the given byte slice.
func Read(data []byte) (*Level, error) {
	level := &Level{}

	if err := json.Unmarshal(data, level); err != nil {
		return nil, err
	}

	// Parse background color
	if level.BGColorString != "" {
		c, err := parseHexColor(level.BGColorString)
		if err == nil {
			level.BGColor = c
		}
	}
	if level.BGColor == nil {
		level.BGColor = color.RGBA{}
	}

	// Ensure maps are initialized
	if level.CustomFields == nil {
		level.CustomFields = map[string]interface{}{}
	}
	if level.Entities == nil {
		level.Entities = map[string][]*Entity{}
	}

	// Ensure entity custom fields are initialized
	for _, entities := range level.Entities {
		for _, entity := range entities {
			if entity.CustomFields == nil {
				entity.CustomFields = map[string]interface{}{}
			}
		}
	}

	return level, nil
}

var errInvalidFormat = errors.New("invalid hex color format")

func parseHexColor(s string) (color.RGBA, error) {
	c := color.RGBA{A: 0xff}

	if len(s) == 0 || s[0] != '#' {
		return c, errInvalidFormat
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		return c, errInvalidFormat
	}

	return c, nil
}
