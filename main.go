package main

import (
	"fmt"
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const FPS = 60

var (
	screenWidth    int32      = 800
	screenHeight   int32      = 450
	center         rl.Vector2 = rl.Vector2{X: float32(screenWidth / 2), Y: float32(screenHeight / 2)}
	headerPanel    int32      = 50
	statusPanel    int32      = 50
	gameRegionXMin int32      = -20
	gameRegionXMax int32      = screenWidth + 20
	gameRegionYMin int32      = headerPanel - 20
	gameRegionYMax int32      = screenHeight - statusPanel + 20
	paused         bool       = false
	noPlayer       bool       = true
	youDied        bool       = false
	youWin         bool       = false
	waves          []int      = []int{100, 20, 30, 40, 50, 40, 30, 20, 10, 100}
	wave           int        = 0

	shootCooldown      float32 = 0.02
	shootTimer         float32 = 0
	enemyMoveCooldown  float32 = 1
	enemyMoveTimer     float32 = 0
	waveCooldown       float32 = 5
	waveTimer          float32 = 0
	invurnableTimer    float32 = 0
	invurnableCooldown float32 = 2
	flashing           bool    = false
)

type Object struct {
	Id           int
	Type         ObjectKind
	State        State
	Health       int
	Size         float32
	Speed        int
	Damage       int
	Position     rl.Vector2
	Orientation  float32
	TurnSpeed    float32
	NextPosition rl.Vector2
}

type ObjectKind int

const (
	Null ObjectKind = iota
	Player
	Bullet
	EnemyBullet
	EnemySquare
	EnemyPentagon
	EnemyHexagon
)

type State int

const (
	Still State = iota
	Moving
	Invurnable
)

const MAX_OBJECTS = 100000

var Objects = [MAX_OBJECTS](*Object){}

func initObjects() {
	for i := range MAX_OBJECTS {
		Objects[i] = &Object{Type: Null}
	}
}

func allocObj(obj *Object) int {
	for i, o := range Objects[1:] {
		if (*o).Type == Null {
			if i+1 < len(Objects) {
				obj.Id = i + 1
				Objects[i+1] = obj
				return i + 1
			}
		}
	}

	return 0
}

func free(obj *Object) {
	if obj.Id > 0 && obj.Id < len(Objects) {
		Objects[obj.Id] = &Object{}
	}
}

func getRandomPosition() rl.Vector2 {
	return rl.Vector2{
		X: float32(rl.GetRandomValue(gameRegionXMin, gameRegionXMax)),
		Y: float32(rl.GetRandomValue(gameRegionYMin, gameRegionYMax)),
	}
}

func getRandomPositionEdge() rl.Vector2 {
	randSide := Direction(rl.GetRandomValue(0, 3))
	switch randSide {
	case top:
		return rl.Vector2{
			X: float32(rl.GetRandomValue(gameRegionXMin, gameRegionXMax)),
			Y: float32(gameRegionYMin),
		}
	case bottom:
		return rl.Vector2{
			X: float32(rl.GetRandomValue(gameRegionXMin, gameRegionXMax)),
			Y: float32(gameRegionYMax),
		}
	case left:
		return rl.Vector2{
			X: float32(gameRegionXMin),
			Y: float32(rl.GetRandomValue(gameRegionYMin, gameRegionYMax)),
		}
	case right:
		return rl.Vector2{
			X: float32(gameRegionXMax),
			Y: float32(rl.GetRandomValue(gameRegionYMin, gameRegionYMax)),
		}
	}
	return rl.Vector2{}
}

func outOfBounds(obj *Object) bool {
	if obj.Position.X < -20 || obj.Position.X > float32(screenWidth)+20 {
		return true
	}

	if obj.Position.Y < -20 || obj.Position.Y > float32(screenHeight)+20 {
		return true
	}

	return false
}

func inViewableRegion(obj *Object) bool {
	if obj.Position.X < 0 || obj.Position.X > float32(screenWidth) {
		return false
	}

	if obj.Position.Y < 0 || obj.Position.Y > float32(screenHeight) {
		return false
	}

	return true
}

type Direction int

const (
	left Direction = iota
	right
	top
	bottom
)

func cleanup() {
	for _, o := range Objects {
		if outOfBounds(o) {
			free(o)
			if o.Type == Player {
				noPlayer = true
			}
		}
	}
}

func render() {
	rl.ClearBackground(rl.RayWhite)

	for _, obj := range Objects {
		if !inViewableRegion(obj) {
			continue
		}

		switch obj.Type {
		case Null:
		case Player:
			if invurnableTimer < invurnableCooldown && !flashing {
				flashing = true
			} else {
				v1 := translate(obj.Position, obj.Orientation, 20)
				v2 := translate(obj.Position, obj.Orientation+120, 10)
				v3 := translate(obj.Position, obj.Orientation+240, 10)
				rl.DrawCircleV(v1, 2, rl.Green)
				rl.DrawCircleV(v2, 2, rl.Green)
				rl.DrawCircleV(v3, 2, rl.Green)
				rl.DrawTriangleLines(v1, v2, v3, rl.Green)
				flashing = false
			}

		case Bullet:
			rl.DrawCircleV(obj.Position, obj.Size, rl.Black)
		case EnemyBullet:
			rl.DrawCircleV(obj.Position, obj.Size, rl.Red)
		case EnemySquare, EnemyPentagon, EnemyHexagon:
			rl.DrawPolyLines(obj.Position, int32(obj.Type), obj.Size, 0, rl.Orange)
		}
	}

	rl.DrawRectangle(0, 0, screenWidth, headerPanel, rl.Black)
	rl.DrawText("GeoWars", 2, 0, headerPanel, rl.LightGray)
	rl.DrawRectangle(0, screenHeight-statusPanel, screenWidth, statusPanel, rl.Black)
	objects := 0
	for _, o := range Objects {
		if o.Type != Null {
			objects++
		}
	}
	rl.DrawText(
		fmt.Sprintf(
			"Wave: %d/%d | Lives: %d | Objects: %d",
			wave+1,
			len(waves),
			getPlayer().Health,
			objects,
		),
		2,
		screenHeight-statusPanel/2,
		statusPanel/2,
		rl.LightGray,
	)

	if youDied {
		msg := "You Died"
		var fontSize int32 = 20
		textSize := rl.MeasureText(msg, fontSize)
		rl.DrawRectangle(0, headerPanel, screenWidth, screenHeight-statusPanel, rl.Black)
		rl.DrawText(msg, int32(center.X)-textSize/2, int32(center.Y), fontSize, rl.LightGray)
	}

	if youWin {
		msg := "You Won"
		var fontSize int32 = 20
		textSize := rl.MeasureText(msg, fontSize)
		rl.DrawRectangle(0, headerPanel, screenWidth, screenHeight-statusPanel, rl.Black)
		rl.DrawText(msg, int32(center.X)-textSize/2, int32(center.Y), fontSize, rl.LightGray)
	}

	if paused {
		msg := "Paused"
		var fontSize int32 = 20
		textSize := rl.MeasureText(msg, fontSize)
		rl.DrawRectangle(0, headerPanel, screenWidth, screenHeight-statusPanel, rl.ColorAlpha(rl.Black, 0.5))
		rl.DrawText(msg, int32(center.X)-textSize/2, int32(center.Y), fontSize, rl.LightGray)
	}
}

func translate(pt rl.Vector2, ori float32, dist int) rl.Vector2 {
	angle := float64(rl.Deg2rad * ori)
	return rl.Vector2{
		X: pt.X + float32(math.Cos(angle))*float32(dist),
		Y: pt.Y + float32(math.Sin(angle))*float32(dist),
	}
}

func moveForward(obj *Object) {
	angle := float64(rl.Deg2rad * obj.Orientation)
	obj.Position.X += float32(math.Cos(angle)) * float32(obj.Speed)
	obj.Position.Y += float32(math.Sin(angle)) * float32(obj.Speed)
}

func rotate(player *Object, dir Direction) {
	switch dir {
	case left:
		player.Orientation -= player.TurnSpeed
		if player.Orientation < 0 {
			player.Orientation += 360
		}
	case right:
		player.Orientation += player.TurnSpeed
		if player.Orientation > 360 {
			player.Orientation -= 360
		}
	}
}

func getPlayer() *Object {
	for _, o := range Objects {
		if o.Type == Player {
			return o
		}
	}

	return Objects[0]
}

func isEnemy(obj *Object) bool {
	switch obj.Type {
	case EnemySquare, EnemyPentagon, EnemyHexagon, EnemyBullet:
		return true
	default:
		return false
	}
}

func ai() {
	player := getPlayer()

	for _, o := range Objects {
		switch o.Type {
		case EnemyBullet:
			moveForward(o)
		case EnemyHexagon:
			if enemyMoveTimer > enemyMoveCooldown && o.State == Still {
				o.NextPosition = player.Position
				o.State = Moving
			} else if enemyMoveTimer > enemyMoveCooldown && o.State == Moving {
				o.State = Still
			} else if o.State == Moving {
				o.Position = rl.Vector2MoveTowards(o.Position, o.NextPosition, float32(o.Speed))
			}
		case EnemySquare:
			o.Position = rl.Vector2MoveTowards(o.Position, player.Position, float32(o.Speed))
		case EnemyPentagon:
			o.Position = rl.Vector2MoveTowards(o.Position, player.Position, float32(o.Speed))
			if rand.Float32() < 0.005 {
				v := rl.Vector2Normalize(rl.Vector2Subtract(o.Position, player.Position))
				b := Object{
					Type:        EnemyBullet,
					Health:      1,
					Speed:       2,
					Position:    o.Position,
					Size:        5,
					State:       Moving,
					Orientation: float32(math.Atan(float64(v.Y/v.X))) * rl.Rad2deg,
				}
				allocObj(&b)
			}
		}
	}

	if enemyMoveTimer > enemyMoveCooldown {
		enemyMoveTimer = 0
	}
}

func physics() {
	dt := float32(1.0 / FPS)
	shootTimer += dt
	enemyMoveTimer += dt
	invurnableTimer += dt

	for _, o := range Objects {
		switch o.Type {
		case Player:
			for _, e := range Objects {
				if isEnemy(e) && rl.CheckCollisionCircles(o.Position, o.Size, e.Position, e.Size) && invurnableTimer > invurnableCooldown {
					free(e)
					if o.Health == 0 {
						youDied = true
						return
					}
					o.Health -= 1
					o.Position = getRandomPosition()
					invurnableTimer = 0
					break
				}
			}
		case Bullet:
			moveForward(o)
			for _, e := range Objects {
				if isEnemy(e) && rl.CheckCollisionCircles(o.Position, o.Size, e.Position, e.Size) {
					free(e)
					free(o)
					break
				}
			}
		}
	}
}

func spawnWave() {
	if waveTimer < waveCooldown {
		waveTimer += float32(1.0 / FPS)
		return
	}

	for range waves[wave] {
		sides := rl.GetRandomValue(4, 6)
		speed := 1
		if sides == 6 {
			speed *= 2
		}
		e := Object{
			Type:     ObjectKind(sides),
			Health:   1,
			Speed:    speed,
			Position: getRandomPositionEdge(),
			Size:     20,
		}
		allocObj(&e)
	}

	waveTimer = 0
	wave += 1
}

func checkEnd() {
	if wave >= len(waves) {
		for _, o := range Objects {
			if o.Type == EnemyHexagon || o.Type == EnemyPentagon || o.Type == EnemySquare {
				return
			}
			youWin = true
		}
	}
}

func onlyOnePlayer() bool {
	c := 0
	for _, o := range Objects {
		if o.Type == Player {
			c++
		}
		if c > 1 {
			return false
		}
	}
	return true
}

func main() {
	rl.InitWindow(screenWidth, screenHeight, "GeoWars")
	rl.SetExitKey(rl.KeyEscape)

	rl.SetTargetFPS(FPS)

	initObjects()
	player := Object{
		Type:        Player,
		Health:      3,
		Speed:       10,
		Damage:      1,
		Position:    center,
		Orientation: 270,
		TurnSpeed:   5,
		Size:        30,
	}
	allocObj(&player)
	noPlayer = false

	for !rl.WindowShouldClose() {
		if !onlyOnePlayer() {
			panic("Expected Only 1 Player")
		}

		rl.BeginDrawing()

		// inputs
		if rl.IsKeyPressed(rl.KeyP) && !paused {
			paused = true
		} else if rl.IsKeyPressed(rl.KeyP) {
			paused = false
		}

		if !paused {
			if rl.IsKeyPressed(rl.KeyR) && noPlayer {
				player = Object{
					Type:        Player,
					Health:      3,
					Speed:       10,
					Damage:      1,
					Position:    getRandomPosition(),
					Orientation: 0,
					TurnSpeed:   5,
					Size:        30,
				}
				allocObj(&player)
				noPlayer = false
			}

			if rl.IsKeyDown(rl.KeySpace) {
				if shootTimer > shootCooldown && !noPlayer {
					b := Object{
						Type:        Bullet,
						Health:      1,
						Speed:       5,
						Position:    translate(player.Position, player.Orientation, 25),
						Orientation: player.Orientation,
						Size:        5,
						State:       Moving,
					}
					allocObj(&b)
					shootTimer = 0
				}
			}

			if rl.IsKeyDown(rl.KeyW) {
				moveForward(&player)
			}

			if rl.IsKeyDown(rl.KeyA) {
				rotate(&player, left)
			}

			if rl.IsKeyDown(rl.KeyD) {
				rotate(&player, right)
			}

			// updates
			spawnWave()
			ai()
			physics()
			cleanup()
			checkEnd()
		}

		render()

		rl.EndDrawing()
	}

	free(&player)
	rl.CloseWindow()
}
