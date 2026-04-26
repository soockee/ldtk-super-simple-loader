package ldtkgo

import (
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

// testWorldDir is rooted one level above the .ldtk file so that
// LoadWorld("pong.ldtk", fs) resolves the standard simplified export path:
//
//	pong/simplified/level_0/data.json
const testWorldDir = "examples/pong/ldtk"

// testLowLevelDir is rooted at the pong project folder for low-level tests
// that use Open/OpenProject directly.
const testLowLevelDir = "examples/pong/ldtk/pong"

// loadTestWorld is a helper that calls LoadWorld and fails the test on error.
func loadTestWorld(t *testing.T) *World {
	t.Helper()
	world, err := LoadWorld("pong.ldtk", os.DirFS(testWorldDir))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	return world
}

// =============================================================================
// LoadWorld — single-call entry point
// =============================================================================

func TestLoadWorld(t *testing.T) {
	world := loadTestWorld(t)

	if world.Project == nil {
		t.Fatal("Project is nil")
	}
	if len(world.Levels) != 2 {
		t.Fatalf("Levels = %d, want 2", len(world.Levels))
	}
}

func TestLoadWorldNonexistent(t *testing.T) {
	_, err := LoadWorld("nonexistent.ldtk", os.DirFS(testWorldDir))
	if err == nil {
		t.Error("LoadWorld(nonexistent) should return error")
	}
}

// =============================================================================
// World.Level — lookup by identifier
// =============================================================================

func TestWorldLevel(t *testing.T) {
	world := loadTestWorld(t)

	level := world.Level("level_0")
	if level == nil {
		t.Fatal("Level(level_0) = nil")
	}
	if level.Identifier != "level_0" {
		t.Errorf("Identifier = %q", level.Identifier)
	}
	if level.Width != 1280 || level.Height != 720 {
		t.Errorf("dimensions = %dx%d, want 1280x720", level.Width, level.Height)
	}
}

func TestWorldLevelNotFound(t *testing.T) {
	world := loadTestWorld(t)
	if world.Level("nonexistent") != nil {
		t.Error("Level(nonexistent) should be nil")
	}
}

// =============================================================================
// World.Tileset — lookup by identifier
// =============================================================================

func TestWorldTileset(t *testing.T) {
	world := loadTestWorld(t)

	ts := world.Tileset("ball")
	if ts == nil {
		t.Fatal("Tileset(ball) = nil")
	}
	if ts.Image == nil {
		t.Error("ball tileset image not loaded")
	}
	if ts.PxWid != 32 || ts.PxHei != 32 {
		t.Errorf("ball tileset = %dx%d, want 32x32", ts.PxWid, ts.PxHei)
	}

	if world.Tileset("nonexistent") != nil {
		t.Error("Tileset(nonexistent) should be nil")
	}
}

// =============================================================================
// World.EntityDef — lookup by identifier
// =============================================================================

func TestWorldEntityDef(t *testing.T) {
	world := loadTestWorld(t)

	def := world.EntityDef("ball")
	if def == nil {
		t.Fatal("EntityDef(ball) = nil")
	}
	if def.RenderMode != "Tile" {
		t.Errorf("RenderMode = %q, want Tile", def.RenderMode)
	}
	if def.TileRect == nil {
		t.Fatal("TileRect = nil")
	}

	if world.EntityDef("nonexistent") != nil {
		t.Error("EntityDef(nonexistent) should be nil")
	}
}

// =============================================================================
// Level.Entity — single entity shorthand
// =============================================================================

func TestLevelEntity(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	ball := level.Entity("ball")
	if ball == nil {
		t.Fatal("Entity(ball) = nil")
	}
	if ball.ID != "ball" {
		t.Errorf("ID = %q", ball.ID)
	}
	if ball.X != 560 || ball.Y != 320 {
		t.Errorf("position = (%d,%d), want (560,320)", ball.X, ball.Y)
	}

	if level.Entity("nonexistent") != nil {
		t.Error("Entity(nonexistent) should be nil")
	}
}

// =============================================================================
// Level.EntitiesByID — all instances of a type
// =============================================================================

func TestLevelEntitiesByID(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	players := level.EntitiesByID("player_left")
	if len(players) != 1 {
		t.Fatalf("player_left count = %d, want 1", len(players))
	}
	if players[0].X != 80 || players[0].Y != 336 {
		t.Errorf("player_left pos = (%d,%d), want (80,336)", players[0].X, players[0].Y)
	}

	if level.EntitiesByID("nonexistent") != nil {
		t.Error("EntitiesByID(nonexistent) should be nil")
	}
}

// =============================================================================
// Level.EntityByIID — lookup by unique instance ID
// =============================================================================

func TestLevelEntityByIID(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	entity := level.EntityByIID("f6111450-21a0-11f1-91de-b57b33ae5f6f")
	if entity == nil {
		t.Fatal("EntityByIID returned nil")
	}
	if entity.ID != "player_right" {
		t.Errorf("ID = %q, want player_right", entity.ID)
	}

	if level.EntityByIID("nonexistent-iid") != nil {
		t.Error("EntityByIID(nonexistent) should be nil")
	}
}

// =============================================================================
// Level.AllEntities — flat list
// =============================================================================

func TestLevelAllEntities(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	all := level.AllEntities()
	if len(all) != 3 {
		t.Errorf("AllEntities = %d, want 3", len(all))
	}

	ids := map[string]bool{}
	for _, e := range all {
		ids[e.ID] = true
	}
	for _, want := range []string{"ball", "player_left", "player_right"} {
		if !ids[want] {
			t.Errorf("AllEntities missing %q", want)
		}
	}
}

// =============================================================================
// Entity convenience methods
// =============================================================================

func TestEntityColorRGBA(t *testing.T) {
	world := loadTestWorld(t)
	ball := world.Level("level_0").Entity("ball")

	// ball.Color = 16777215 = 0xFFFFFF
	c := ball.ColorRGBA()
	if c.R != 0xFF || c.G != 0xFF || c.B != 0xFF || c.A != 0xFF {
		t.Errorf("ColorRGBA = %+v, want white", c)
	}

	player := world.Level("level_0").Entity("player_left")
	// player_left.Color = 16711748 = 0xFF0044
	pc := player.ColorRGBA()
	if pc.R != 0xFF || pc.G != 0x00 || pc.B != 0x44 {
		t.Errorf("player_left ColorRGBA = %+v, want {0xFF,0x00,0x44}", pc)
	}
}

func TestEntityPos(t *testing.T) {
	world := loadTestWorld(t)
	ball := world.Level("level_0").Entity("ball")

	x, y := ball.Pos()
	if x != 560 || y != 320 {
		t.Errorf("Pos = (%d,%d), want (560,320)", x, y)
	}
}

func TestEntitySize(t *testing.T) {
	world := loadTestWorld(t)
	ball := world.Level("level_0").Entity("ball")

	w, h := ball.Size()
	if w != 32 || h != 32 {
		t.Errorf("Size = (%d,%d), want (32,32)", w, h)
	}
}

func TestEntityRect(t *testing.T) {
	world := loadTestWorld(t)
	ball := world.Level("level_0").Entity("ball")

	r := ball.Rect()
	want := image.Rect(560, 320, 592, 352)
	if r != want {
		t.Errorf("Rect = %v, want %v", r, want)
	}
}

// =============================================================================
// Entity.SubImage — sprite extraction from tileset
// =============================================================================

func TestEntitySubImage(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	// Ball has a tile rect → SubImage returns the sprite
	ball := level.Entity("ball")
	sub := ball.SubImage()
	if sub == nil {
		t.Fatal("ball.SubImage() = nil")
	}
	if sub.Bounds().Dx() != 32 || sub.Bounds().Dy() != 32 {
		t.Errorf("ball SubImage = %dx%d, want 32x32", sub.Bounds().Dx(), sub.Bounds().Dy())
	}

	// Player has no tile rect → SubImage returns nil
	player := level.Entity("player_right")
	if player.SubImage() != nil {
		t.Error("player.SubImage() should be nil")
	}
}

func TestEntitySubImageNoLink(t *testing.T) {
	entity := &Entity{ID: "test"}
	if entity.SubImage() != nil {
		t.Error("SubImage on unlinked entity should be nil")
	}
}

// =============================================================================
// Entity.Def / Entity.Tileset — linking
// =============================================================================

func TestEntityDefLinked(t *testing.T) {
	world := loadTestWorld(t)
	ball := world.Level("level_0").Entity("ball")

	if ball.Def == nil {
		t.Fatal("Def is nil")
	}
	if ball.Def.Identifier != "ball" {
		t.Errorf("Def.Identifier = %q", ball.Def.Identifier)
	}
	if ball.Tileset == nil {
		t.Fatal("Tileset is nil")
	}
	if ball.Tileset.Identifier != "ball" {
		t.Errorf("Tileset.Identifier = %q", ball.Tileset.Identifier)
	}
}

func TestEntityDefLinkedNoTileset(t *testing.T) {
	world := loadTestWorld(t)
	player := world.Level("level_0").Entity("player_right")

	if player.Def == nil {
		t.Fatal("Def is nil")
	}
	if player.Tileset != nil {
		t.Error("player.Tileset should be nil")
	}
}

// =============================================================================
// Entity tags — inherited from EntityDef via LinkProject
// =============================================================================

func TestEntityTagsInherited(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	ball := level.Entity("ball")
	if len(ball.Tags) != 1 || ball.Tags[0] != "ball" {
		t.Errorf("ball.Tags = %v, want [ball]", ball.Tags)
	}

	playerL := level.Entity("player_left")
	if len(playerL.Tags) != 1 || playerL.Tags[0] != "player" {
		t.Errorf("player_left.Tags = %v, want [player]", playerL.Tags)
	}

	playerR := level.Entity("player_right")
	if len(playerR.Tags) != 1 || playerR.Tags[0] != "player" {
		t.Errorf("player_right.Tags = %v, want [player]", playerR.Tags)
	}
}

func TestEntityDefTags(t *testing.T) {
	world := loadTestWorld(t)

	ballDef := world.EntityDef("ball")
	if len(ballDef.Tags) != 1 || ballDef.Tags[0] != "ball" {
		t.Errorf("ball EntityDef.Tags = %v, want [ball]", ballDef.Tags)
	}

	playerDef := world.EntityDef("player_left")
	if len(playerDef.Tags) != 1 || playerDef.Tags[0] != "player" {
		t.Errorf("player_left EntityDef.Tags = %v, want [player]", playerDef.Tags)
	}
}

func TestEntitiesByTag(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	players := level.EntitiesByTag("player")
	if len(players) != 2 {
		t.Fatalf("EntitiesByTag(player) = %d, want 2", len(players))
	}
	ids := map[string]bool{}
	for _, e := range players {
		ids[e.ID] = true
	}
	if !ids["player_left"] || !ids["player_right"] {
		t.Errorf("player IDs = %v, want player_left + player_right", ids)
	}

	balls := level.EntitiesByTag("ball")
	if len(balls) != 1 || balls[0].ID != "ball" {
		t.Errorf("EntitiesByTag(ball) = %v", balls)
	}

	none := level.EntitiesByTag("nonexistent")
	if len(none) != 0 {
		t.Errorf("EntitiesByTag(nonexistent) = %d, want 0", len(none))
	}
}

func TestEntityTagsUnlinked(t *testing.T) {
	entity := &Entity{ID: "test"}
	if len(entity.Tags) != 0 {
		t.Errorf("unlinked entity.Tags = %v, want empty", entity.Tags)
	}
}

// =============================================================================
// Level background — color and image
// =============================================================================

func TestLevelBGColor(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	if level.BGColorString != "#696A79" {
		t.Errorf("BGColorString = %q", level.BGColorString)
	}
	rgba, ok := level.BGColor.(color.RGBA)
	if !ok {
		t.Fatalf("BGColor type = %T, want color.RGBA", level.BGColor)
	}
	if rgba.R != 0x69 || rgba.G != 0x6A || rgba.B != 0x79 {
		t.Errorf("BGColor = %+v", rgba)
	}
}

func TestLevelBGImage(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	if level.BGRelPath != "tilesets/sky_background.png" {
		t.Errorf("BGRelPath = %q", level.BGRelPath)
	}
	if level.BGImage == nil {
		t.Error("BGImage is nil")
	}
}

// =============================================================================
// Level.LoadedLayers — layer images loaded by LoadWorld
// =============================================================================

func TestLevelLoadedLayers(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	if len(level.LoadedLayers) < 2 {
		t.Fatalf("LoadedLayers = %d, want >= 2", len(level.LoadedLayers))
	}

	var hasBG, hasTiles, hasComposite bool
	for _, layer := range level.LoadedLayers {
		if layer.Image == nil {
			t.Errorf("layer %q has nil image", layer.Name)
		}
		switch layer.Type {
		case LayerBackground:
			hasBG = true
		case LayerTiles:
			hasTiles = true
		case LayerComposite:
			hasComposite = true
		}
	}

	if !hasBG {
		t.Error("missing background layer")
	}
	if !hasTiles {
		t.Error("missing tiles layer")
	}
	if !hasComposite {
		t.Error("missing composite layer")
	}

	// Order: bg first, composite last
	if level.LoadedLayers[0].Type != LayerBackground {
		t.Error("first layer should be background")
	}
	if level.LoadedLayers[len(level.LoadedLayers)-1].Type != LayerComposite {
		t.Error("last layer should be composite")
	}
}

// =============================================================================
// Level.Layers — PNG filename list from data.json
// =============================================================================

func TestLevelLayerNames(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	if len(level.Layers) != 1 {
		t.Errorf("Layers = %v, want 1 entry", level.Layers)
	}
	if level.Layers[0] != "collision_test.png" {
		t.Errorf("Layers[0] = %q, want collision_test.png", level.Layers[0])
	}
}

// =============================================================================
// Entity.Data — user-attached data
// =============================================================================

func TestEntityUserData(t *testing.T) {
	world := loadTestWorld(t)
	ball := world.Level("level_0").Entity("ball")

	if ball.Data != nil {
		t.Error("Data should be nil initially")
	}

	ball.Data = "my game object"
	if ball.Data != "my game object" {
		t.Error("Data round-trip failed")
	}
}

// =============================================================================
// Custom fields initialization
// =============================================================================

func TestCustomFieldsInitialized(t *testing.T) {
	level, err := Read([]byte(`{"identifier":"test","entities":{"e":[{"id":"e","iid":"1"}]}}`))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if level.CustomFields == nil {
		t.Error("level.CustomFields is nil")
	}
	if level.Entities["e"][0].CustomFields == nil {
		t.Error("entity.CustomFields is nil")
	}
}

// =============================================================================
// Tileset images loaded
// =============================================================================

func TestTilesetImagesLoaded(t *testing.T) {
	world := loadTestWorld(t)

	for _, ts := range world.Project.Tilesets {
		if ts.Image == nil {
			t.Errorf("tileset %q image not loaded", ts.Identifier)
			continue
		}
		bounds := ts.Image.Bounds()
		if bounds.Dx() != ts.PxWid || bounds.Dy() != ts.PxHei {
			t.Errorf("tileset %q image = %dx%d, want %dx%d",
				ts.Identifier, bounds.Dx(), bounds.Dy(), ts.PxWid, ts.PxHei)
		}
	}
}

// =============================================================================
// Low-level API — kept for internal coverage
// =============================================================================

func TestReadLevel(t *testing.T) {
	data, err := os.ReadFile(testLowLevelDir + "/simplified/level_0/data.json")
	if err != nil {
		t.Fatalf("reading data.json: %v", err)
	}
	level, err := Read(data)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if level.Identifier != "level_0" {
		t.Errorf("Identifier = %q", level.Identifier)
	}
}

func TestOpenLevel(t *testing.T) {
	level, err := Open("simplified/level_0/data.json", os.DirFS(testLowLevelDir))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if level.Identifier != "level_0" {
		t.Errorf("Identifier = %q", level.Identifier)
	}
}

func TestParseHexColorShort(t *testing.T) {
	c, err := parseHexColor("#F80")
	if err != nil {
		t.Fatalf("parseHexColor: %v", err)
	}
	if c.R != 0xFF || c.G != 0x88 || c.B != 0x00 {
		t.Errorf("got %+v, want {R:0xFF, G:0x88, B:0x00}", c)
	}
}

func TestParseHexColorInvalid(t *testing.T) {
	for _, s := range []string{"", "696A79", "#12", "#1234567"} {
		_, err := parseHexColor(s)
		if err == nil {
			t.Errorf("parseHexColor(%q) should return error", s)
		}
	}
}

// --- Error cases ---

func TestReadInvalidJSON(t *testing.T) {
	if _, err := Read([]byte(`{invalid`)); err == nil {
		t.Error("Read(invalid) should return error")
	}
}

func TestReadProjectInvalidJSON(t *testing.T) {
	if _, err := ReadProject([]byte(`{invalid`)); err == nil {
		t.Error("ReadProject(invalid) should return error")
	}
}

func TestOpenNonexistentFile(t *testing.T) {
	if _, err := Open("nonexistent.json", os.DirFS(testLowLevelDir)); err == nil {
		t.Error("Open(nonexistent) should return error")
	}
}

func TestOpenProjectNonexistentFile(t *testing.T) {
	if _, err := OpenProject("nonexistent.ldtk", os.DirFS(testLowLevelDir)); err == nil {
		t.Error("OpenProject(nonexistent) should return error")
	}
}

func TestLoadBGImageNoBGRelPath(t *testing.T) {
	level := &Level{}
	if err := level.LoadBGImage(".", os.DirFS(testLowLevelDir)); err != nil {
		t.Fatalf("LoadBGImage with empty BGRelPath: %v", err)
	}
	if level.BGImage != nil {
		t.Error("BGImage should be nil when BGRelPath is empty")
	}
}

// =============================================================================
// Full integration workflow — the developer-facing one-liner
// =============================================================================

func TestFullWorkflow(t *testing.T) {
	// One call loads everything
	world := loadTestWorld(t)
	level := world.Level("level_0")

	// Get entities by name
	ball := level.Entity("ball")
	playerL := level.Entity("player_left")
	playerR := level.Entity("player_right")

	// Positions come from LDtk
	if x, y := ball.Pos(); x != 560 || y != 320 {
		t.Errorf("ball.Pos = (%d,%d)", x, y)
	}
	if w, h := ball.Size(); w != 32 || h != 32 {
		t.Errorf("ball.Size = (%d,%d)", w, h)
	}

	// Ball has a sprite from its tileset
	if ball.SubImage() == nil {
		t.Error("ball sprite is nil")
	}

	// Players render as colored rectangles
	if playerL.SubImage() != nil {
		t.Error("player_left should not have a sprite")
	}
	if playerR.SubImage() != nil {
		t.Error("player_right should not have a sprite")
	}

	// Colors are accessible as RGBA
	lc := playerL.ColorRGBA()
	if lc.R != 0xFF || lc.G != 0x00 || lc.B != 0x44 {
		t.Errorf("player_left color = %+v", lc)
	}

	// Background and layers are loaded
	if level.BGImage == nil {
		t.Error("BGImage not loaded")
	}
	if len(level.LoadedLayers) == 0 {
		t.Error("no layers loaded")
	}

	// User data can be attached
	ball.Data = struct{ Speed float64 }{Speed: 4.0}
	if ball.Data.(struct{ Speed float64 }).Speed != 4.0 {
		t.Error("user data round-trip failed")
	}

	// Tileset accessible by name
	if world.Tileset("ball") == nil {
		t.Error("Tileset(ball) not found")
	}

	// Entity def accessible by name
	if world.EntityDef("ball") == nil {
		t.Error("EntityDef(ball) not found")
	}
}

// =============================================================================
// IntGrid — loaded from CSV in simplified export
// =============================================================================

func TestIntGridLoaded(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	ig := level.IntGrid("collision_test")
	if ig == nil {
		t.Fatal("IntGrid(collision_test) = nil")
	}
	if ig.Name != "collision_test" {
		t.Errorf("Name = %q", ig.Name)
	}
	// 1280/8 = 160 columns, 720/8 = 90 rows
	if ig.Width != 160 {
		t.Errorf("Width = %d, want 160", ig.Width)
	}
	if ig.Height != 90 {
		t.Errorf("Height = %d, want 90", ig.Height)
	}
}

func TestIntGridAt(t *testing.T) {
	world := loadTestWorld(t)
	ig := world.Level("level_0").IntGrid("collision_test")

	// Row 0, col 0 should be 1 (border wall)
	if v := ig.At(0, 0); v != 1 {
		t.Errorf("At(0,0) = %d, want 1", v)
	}
	// Center of the grid should be 0 (empty playfield)
	if v := ig.At(80, 45); v != 0 {
		t.Errorf("At(80,45) = %d, want 0", v)
	}
	// Out of bounds returns 0
	if v := ig.At(-1, 0); v != 0 {
		t.Errorf("At(-1,0) = %d, want 0", v)
	}
	if v := ig.At(160, 0); v != 0 {
		t.Errorf("At(160,0) = %d, want 0", v)
	}
}

func TestIntGridAtPx(t *testing.T) {
	world := loadTestWorld(t)
	ig := world.Level("level_0").IntGrid("collision_test")

	// (0,0) → border wall, value 1
	if v := ig.AtPx(0, 0); v != 1 {
		t.Errorf("AtPx(0,0) = %d, want 1", v)
	}
	// center of level → empty, value 0
	if v := ig.AtPx(640, 360); v != 0 {
		t.Errorf("AtPx(640,360) = %d, want 0", v)
	}
}

func TestIntGridDef(t *testing.T) {
	world := loadTestWorld(t)
	ig := world.Level("level_0").IntGrid("collision_test")

	if ig.Def == nil {
		t.Fatal("Def is nil")
	}
	if ig.Def.Identifier != "collision_test" {
		t.Errorf("Def.Identifier = %q", ig.Def.Identifier)
	}
	if ig.Def.GridSize != 8 {
		t.Errorf("Def.GridSize = %d, want 8", ig.Def.GridSize)
	}
	if ig.Def.Type != "IntGrid" {
		t.Errorf("Def.Type = %q", ig.Def.Type)
	}
	if len(ig.Def.IntGridValues) != 1 {
		t.Fatalf("IntGridValues = %d, want 1", len(ig.Def.IntGridValues))
	}
	if ig.Def.IntGridValues[0].Value != 1 {
		t.Errorf("IntGridValues[0].Value = %d, want 1", ig.Def.IntGridValues[0].Value)
	}
}

func TestIntGridNotFound(t *testing.T) {
	world := loadTestWorld(t)
	level := world.Level("level_0")

	if level.IntGrid("nonexistent") != nil {
		t.Error("IntGrid(nonexistent) should be nil")
	}
}

func TestIntGridAtPxNoLink(t *testing.T) {
	ig := &IntGrid{Width: 2, Height: 2, Grid: [][]int{{1, 0}, {0, 1}}}
	// No Def → AtPx returns 0
	if v := ig.AtPx(0, 0); v != 0 {
		t.Errorf("AtPx without Def = %d, want 0", v)
	}
	// But At still works
	if v := ig.At(0, 0); v != 1 {
		t.Errorf("At(0,0) = %d, want 1", v)
	}
}

func TestParseIntGridCSV(t *testing.T) {
	csv := "0,1,0,\n1,0,1,\n"
	ig, err := parseIntGridCSV("test", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseIntGridCSV: %v", err)
	}
	if ig.Width != 3 || ig.Height != 2 {
		t.Fatalf("size = %dx%d, want 3x2", ig.Width, ig.Height)
	}
	if ig.At(1, 0) != 1 {
		t.Errorf("At(1,0) = %d, want 1", ig.At(1, 0))
	}
	if ig.At(0, 1) != 1 {
		t.Errorf("At(0,1) = %d, want 1", ig.At(0, 1))
	}
}
