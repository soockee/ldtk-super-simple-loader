package main

import (
	"image"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	ldtkgo "github.com/soockee/ldtk-super-simple-loader"
)

const (
	ldtkDir = "ldtk"

	ballSpeed      = 4.0
	paddleSpeed    = 5.0
	maxBounceAngle = math.Pi / 3 // 60 degrees
)

// Paddle represents a player paddle.
type Paddle struct {
	entity *ldtkgo.Entity
	x, y   float64
	w, h   float64
	sprite *ebiten.Image
	score  int
}

// Ball represents the game ball.
type Ball struct {
	entity *ldtkgo.Entity
	x, y   float64
	w, h   float64
	vx, vy float64
	sprite *ebiten.Image
}

// Game holds all game state, loaded from LDtk.
type Game struct {
	screenW, screenH int

	world *ldtkgo.World
	level *ldtkgo.Level

	bgImage     *ebiten.Image
	layerImages []*ebiten.Image

	left  Paddle
	right Paddle
	ball  Ball

	// levelKeys maps ebiten keys 1-9 to level indices.
	levelKeys []ebiten.Key
}

func main() {
	// One call loads everything: project, tilesets, levels, layers, BG images
	world, err := ldtkgo.LoadWorld("pong.ldtk", os.DirFS(ldtkDir))
	if err != nil {
		log.Fatalf("loading world: %v", err)
	}

	game := &Game{
		world: world,
		levelKeys: []ebiten.Key{
			ebiten.Key1, ebiten.Key2, ebiten.Key3,
			ebiten.Key4, ebiten.Key5, ebiten.Key6,
			ebiten.Key7, ebiten.Key8, ebiten.Key9,
		},
	}

	game.loadLevel(0)

	ebiten.SetWindowSize(game.screenW, game.screenH)
	ebiten.SetWindowTitle("Pong — ldtk-super-simple-loader example")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// loadLevel switches the game to the level at the given index in world.Levels.
func (g *Game) loadLevel(index int) {
	if index < 0 || index >= len(g.world.Levels) {
		return
	}

	level := g.world.Levels[index]
	g.level = level
	g.screenW = level.Width
	g.screenH = level.Height

	// Convert background image to ebiten image
	g.bgImage = nil
	if level.BGImage != nil {
		g.bgImage = ebiten.NewImageFromImage(level.BGImage)
	}

	// Convert layer images to ebiten images
	g.layerImages = nil
	for _, l := range level.LoadedLayers {
		g.layerImages = append(g.layerImages, ebiten.NewImageFromImage(l.Image))
	}

	// Initialize paddles from entities tagged "player"
	g.left = Paddle{}
	g.right = Paddle{}
	players := level.EntitiesByTag("player")
	for _, p := range players {
		paddle := newPaddle(p)
		switch p.ID {
		case "player_left":
			g.left = paddle
		case "player_right":
			g.right = paddle
		}
	}

	// Initialize ball from entity tagged "ball"
	balls := level.EntitiesByTag("ball")
	if len(balls) == 0 {
		log.Printf("level %s: no entity tagged 'ball'", level.Identifier)
		return
	}
	ballEntity := balls[0]
	bx, by := ballEntity.Pos()
	bw, bh := ballEntity.Size()
	g.ball = Ball{
		entity: ballEntity,
		x:      float64(bx),
		y:      float64(by),
		w:      float64(bw),
		h:      float64(bh),
		vx:     ballSpeed,
		vy:     ballSpeed,
	}
	// Use tileset SubImage for ball sprite
	if sub := ballEntity.SubImage(); sub != nil {
		g.ball.sprite = ebiten.NewImageFromImage(sub)
	}
}

func newPaddle(entity *ldtkgo.Entity) Paddle {
	x, y := entity.Pos()
	w, h := entity.Size()
	p := Paddle{
		entity: entity,
		x:      float64(x),
		y:      float64(y),
		w:      float64(w),
		h:      float64(h),
	}
	// Use tileset SubImage if the entity has one
	if sub := entity.SubImage(); sub != nil {
		p.sprite = ebiten.NewImageFromImage(sub)
	}
	return p
}

func (g *Game) Update() error {
	// Level switching: keys 1-9
	for i, key := range g.levelKeys {
		if i >= len(g.world.Levels) {
			break
		}
		if ebiten.IsKeyPressed(key) && g.world.Levels[i] != g.level {
			g.loadLevel(i)
			return nil
		}
	}

	// Left paddle: W/S
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.left.y -= paddleSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.left.y += paddleSpeed
	}

	// Right paddle: Up/Down
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.right.y -= paddleSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.right.y += paddleSpeed
	}

	// Clamp paddles to screen
	g.left.y = clamp(g.left.y, 0, float64(g.screenH)-g.left.h)
	g.right.y = clamp(g.right.y, 0, float64(g.screenH)-g.right.h)

	// Move ball
	g.ball.x += g.ball.vx
	g.ball.y += g.ball.vy

	// Ball-wall collision using IntGrid (top/bottom), fallback to screen edges
	collision := g.level.IntGrid("collision_test")
	if collision != nil {
		// Check top edge of ball
		if g.ball.vy < 0 && collision.AtPx(int(g.ball.x+g.ball.w/2), int(g.ball.y)) != 0 {
			g.ball.vy = -g.ball.vy
		}
		// Check bottom edge of ball
		if g.ball.vy > 0 && collision.AtPx(int(g.ball.x+g.ball.w/2), int(g.ball.y+g.ball.h)) != 0 {
			g.ball.vy = -g.ball.vy
		}
	} else {
		// Fallback: hardcoded screen edges
		if g.ball.y <= 0 {
			g.ball.y = 0
			g.ball.vy = -g.ball.vy
		}
		if g.ball.y+g.ball.h >= float64(g.screenH) {
			g.ball.y = float64(g.screenH) - g.ball.h
			g.ball.vy = -g.ball.vy
		}
	}

	// Ball-paddle collision
	if g.ballHitsPaddle(&g.left) {
		g.ball.x = g.left.x + g.left.w
		g.deflectBall(&g.left)
	}
	if g.ballHitsPaddle(&g.right) {
		g.ball.x = g.right.x - g.ball.w
		g.deflectBall(&g.right)
	}

	// Scoring
	if g.ball.x+g.ball.w < 0 {
		g.right.score++
		g.resetBall(1)
	}
	if g.ball.x > float64(g.screenW) {
		g.left.score++
		g.resetBall(-1)
	}

	return nil
}

func (g *Game) deflectBall(p *Paddle) {
	paddleCenter := p.y + p.h/2
	ballCenter := g.ball.y + g.ball.h/2
	relativeIntersect := (paddleCenter - ballCenter) / (p.h / 2)
	angle := relativeIntersect * maxBounceAngle

	speed := math.Sqrt(g.ball.vx*g.ball.vx + g.ball.vy*g.ball.vy)
	dir := 1.0
	if g.ball.vx > 0 {
		dir = -1.0
	}
	g.ball.vx = dir * speed * math.Cos(angle)
	g.ball.vy = -speed * math.Sin(angle)
}

func (g *Game) ballHitsPaddle(p *Paddle) bool {
	return g.ball.x < p.x+p.w &&
		g.ball.x+g.ball.w > p.x &&
		g.ball.y < p.y+p.h &&
		g.ball.y+g.ball.h > p.y
}

func (g *Game) resetBall(dirX float64) {
	x, y := g.ball.entity.Pos()
	g.ball.x = float64(x)
	g.ball.y = float64(y)
	g.ball.vx = dirX * ballSpeed
	g.ball.vy = ballSpeed
}

func (g *Game) Draw(screen *ebiten.Image) {
	// 1. Background image from project
	if g.bgImage != nil {
		op := &ebiten.DrawImageOptions{}
		// Scale to fit level
		bw := float64(g.bgImage.Bounds().Dx())
		bh := float64(g.bgImage.Bounds().Dy())
		op.GeoM.Scale(float64(g.screenW)/bw, float64(g.screenH)/bh)
		screen.DrawImage(g.bgImage, op)
	}

	// 2. Layer images in order (bg layer, tile layers, composite)
	for i, img := range g.layerImages {
		layer := g.level.LoadedLayers[i]
		// Skip background layer if we already drew the project bg
		if layer.Type == ldtkgo.LayerBackground && g.bgImage != nil {
			continue
		}
		// Skip composite — we draw entities on top of tiles
		if layer.Type == ldtkgo.LayerComposite {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(img, op)
	}

	// 3. Draw paddles
	g.drawPaddle(screen, &g.left)
	g.drawPaddle(screen, &g.right)

	// 4. Draw ball
	g.drawBall(screen)

	// 5. Score display
	ebitenutil.DebugPrintAt(screen, scoreText(g.left.score, g.right.score),
		g.screenW/2-40, 10)
}

func (g *Game) drawPaddle(screen *ebiten.Image, p *Paddle) {
	if p.sprite != nil {
		op := &ebiten.DrawImageOptions{}
		// Scale sprite to entity dimensions
		sw := float64(p.sprite.Bounds().Dx())
		sh := float64(p.sprite.Bounds().Dy())
		op.GeoM.Scale(p.w/sw, p.h/sh)
		op.GeoM.Translate(p.x, p.y)
		screen.DrawImage(p.sprite, op)
	} else {
		// No tileset — render as colored rectangle (matches LDtk Rectangle renderMode)
		c := p.entity.ColorRGBA()
		vector.DrawFilledRect(screen, float32(p.x), float32(p.y),
			float32(p.w), float32(p.h), c, false)
	}
}

func (g *Game) drawBall(screen *ebiten.Image) {
	if g.ball.sprite != nil {
		op := &ebiten.DrawImageOptions{}
		sw := float64(g.ball.sprite.Bounds().Dx())
		sh := float64(g.ball.sprite.Bounds().Dy())
		op.GeoM.Scale(g.ball.w/sw, g.ball.h/sh)
		op.GeoM.Translate(g.ball.x, g.ball.y)
		screen.DrawImage(g.ball.sprite, op)
	} else {
		c := g.ball.entity.ColorRGBA()
		vector.DrawFilledRect(screen, float32(g.ball.x), float32(g.ball.y),
			float32(g.ball.w), float32(g.ball.h), c, false)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenW, g.screenH
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func scoreText(left, right int) string {
	return string(rune('0'+left)) + " - " + string(rune('0'+right))
}

// Ensure we don't import image/png init side-effect is registered
// (already done by the library's loadPNG, but ebiten needs it too for NewImageFromImage)
var _ image.Image
