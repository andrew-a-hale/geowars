package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	FPS         = 60
	MAX_OBJECTS = 100000
)

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

type ObjectState int

const (
	None ObjectState = iota
	Still
	Moving
	Invurnable
)

type Object struct {
	Id                 int
	Type               ObjectKind
	State              ObjectState
	Health             int
	Size               float32
	Speed              int
	Damage             int
	Position           rl.Vector2
	Orientation        float32
	TurnSpeed          float32
	NextPosition       rl.Vector2
	ShootTimer         int
	ShootCooldown      int
	InvurnableTimer    int
	InvurnableCooldown int
	MoveTimer          int
	MoveCooldown       int
}

var (
	// UI
	screenWidth    int32      = 800
	screenHeight   int32      = 450
	headerPanel    int32      = 50
	statusPanel    int32      = 50
	outerBorder    int32      = 20
	gameRegionXMin int32      = -outerBorder
	gameRegionXMax int32      = screenWidth + outerBorder
	gameRegionYMin int32      = headerPanel - outerBorder
	gameRegionYMax int32      = screenHeight - statusPanel + outerBorder
	center         rl.Vector2 = rl.Vector2{
		X: float32(screenWidth / 2),
		Y: float32(screenHeight / 2),
	}

	// Game State
	Objects      [MAX_OBJECTS]*Object = [MAX_OBJECTS](*Object){}
	paused       bool                 = false
	youDied      bool                 = false
	youWin       bool                 = false
	waves        []int                = []int{50, 20, 30, 40, 50, 40, 30, 20, 10, 100}
	wave         int                  = 0
	waveCooldown int                  = 5 * FPS
	waveTimer    int                  = 0
)

func initObjects() {
	for i := range MAX_OBJECTS {
		Objects[i] = &Object{Type: Null}
	}
}

func allocObj(obj *Object) int {
	for i, o := range Objects {
		if i == 0 {
			continue
		}

		if (*o).Type == Null {
			if i < len(Objects) {
				obj.Id = i
				Objects[i] = obj
				return i
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

func getRandomPositionEdge() rl.Vector2 {
	randSide := Direction(rl.GetRandomValue(0, 3))
	switch randSide {
	case TOP:
		return rl.Vector2{
			X: float32(rl.GetRandomValue(gameRegionXMin, gameRegionXMax)),
			Y: float32(gameRegionYMin),
		}
	case BOTTOM:
		return rl.Vector2{
			X: float32(rl.GetRandomValue(gameRegionXMin, gameRegionXMax)),
			Y: float32(gameRegionYMax),
		}
	case LEFT:
		return rl.Vector2{
			X: float32(gameRegionXMin),
			Y: float32(rl.GetRandomValue(gameRegionYMin, gameRegionYMax)),
		}
	case RIGHT:
		return rl.Vector2{
			X: float32(gameRegionXMax),
			Y: float32(rl.GetRandomValue(gameRegionYMin, gameRegionYMax)),
		}
	}
	return rl.Vector2{}
}

func outOfBounds(obj *Object) bool {
	if obj.Position.X < -float32(outerBorder) ||
		obj.Position.X > float32(screenWidth)+float32(outerBorder) {
		return true
	}

	if obj.Position.Y < -float32(outerBorder) ||
		obj.Position.Y > float32(screenHeight)+float32(outerBorder) {
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
	LEFT Direction = iota
	RIGHT
	TOP
	BOTTOM
)

func cleanup() {
	for _, o := range Objects {
		if outOfBounds(o) {
			if o.Type == Player {
				o.Health -= 1
				if o.Health < 0 {
					youDied = true
					return
				}
				o.Position = center
				o.InvurnableTimer = 0
				o.State = Invurnable
			} else {
				free(o)
			}
		}
	}
}

func drawPlayer(player *Object) {
	var color rl.Color
	if (player.InvurnableTimer/6)%2 == 0 && player.State == Invurnable {
		color = rl.ColorAlpha(rl.Green, 0.25)
	} else {
		color = rl.Green
	}

	v1 := translate(player.Position, player.Orientation, 20)
	v2 := translate(player.Position, player.Orientation+120, 10)
	v3 := translate(player.Position, player.Orientation+240, 10)
	rl.DrawCircleV(v1, 2, color)
	rl.DrawCircleV(v2, 2, color)
	rl.DrawCircleV(v3, 2, color)
	rl.DrawTriangle(v3, v2, v1, color)
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
			drawPlayer(obj)
		case Bullet:
			rl.DrawCircleV(obj.Position, obj.Size, rl.Black)
		case EnemyBullet:
			rl.DrawCircleV(obj.Position, obj.Size, rl.Red)
		case EnemySquare, EnemyPentagon, EnemyHexagon:
			rl.DrawPolyLines(obj.Position, int32(obj.Type), obj.Size, 0, rl.Orange)
			rl.DrawText(fmt.Sprintf("%d", obj.Id), int32(obj.Position.X), int32(obj.Position.Y), 5, rl.Black)
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
	case LEFT:
		player.Orientation -= player.TurnSpeed
		if player.Orientation < 0 {
			player.Orientation += 360
		}
	case RIGHT:
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

func getRandomViewablePosition() rl.Vector2 {
	return rl.Vector2{
		X: float32(rl.GetRandomValue(gameRegionXMin+outerBorder, gameRegionXMax-outerBorder)),
		Y: float32(rl.GetRandomValue(gameRegionYMin+outerBorder, gameRegionYMax-outerBorder)),
	}
}

func moveToPositionOnDelay(o *Object, pos rl.Vector2) {
	if o.MoveTimer > o.MoveCooldown && o.State == Still {
		o.NextPosition = pos
		o.State = Moving
		o.MoveTimer = 0
	} else if o.MoveTimer > o.MoveCooldown && o.State == Moving {
		o.State = Still
		o.MoveTimer = 0
	} else if o.State == Moving {
		o.Position = rl.Vector2MoveTowards(o.Position, o.NextPosition, float32(o.Speed))
	}
}

func ai() {
	player := getPlayer()

	for _, o := range Objects {
		o.ShootTimer += 1
		o.MoveTimer += 1

		switch o.Type {
		case EnemyBullet:
			moveForward(o)
		case EnemySquare:
			o.Position = rl.Vector2MoveTowards(o.Position, player.Position, float32(o.Speed))
		case EnemyPentagon:
			moveToPositionOnDelay(o, getRandomViewablePosition())
			if o.ShootTimer > o.ShootCooldown {
				v := rl.Vector2Normalize(rl.Vector2Subtract(player.Position, o.Position))
				b := Object{
					Type:        EnemyBullet,
					Speed:       1,
					Position:    o.Position,
					Size:        3,
					State:       Moving,
					Orientation: float32(math.Atan2(float64(v.Y), float64(v.X))) * rl.Rad2deg,
				}
				allocObj(&b)
				o.ShootTimer = 0
			}
		case EnemyHexagon:
			moveToPositionOnDelay(o, player.Position)
		}
	}
}

func physics() {
	for _, o := range Objects {
		switch o.Type {
		case Player:
			o.ShootTimer += 1
			o.InvurnableTimer += 1
			for _, e := range Objects {
				if isEnemy(e) && rl.CheckCollisionCircles(o.Position, o.Size, e.Position, e.Size) {
					free(e)
					if o.InvurnableTimer > o.InvurnableCooldown && o.State == None {
						o.Health -= 1
						if o.Health < 0 {
							youDied = true
							return
						}
						o.Position = center
						o.InvurnableTimer = 0
						o.State = Invurnable
					} else if o.InvurnableTimer > o.InvurnableCooldown && o.State == Invurnable {
						o.State = None
					}

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
		waveTimer += 1
		return
	}

	for range waves[wave] {
		kind := ObjectKind(rl.GetRandomValue(4, 6))
		e := Object{
			Type:     kind,
			Position: getRandomPositionEdge(),
			Size:     20,
		}

		switch kind {
		case EnemySquare:
			e.Speed = 1
			e.State = Moving
		case EnemyPentagon:
			e.Speed = 1
			e.State = Still
			e.ShootTimer = 0
			e.ShootCooldown = 120 + int(rl.GetRandomValue(0, 120))
			e.MoveCooldown = 120
		case EnemyHexagon:
			e.Speed = 2
			e.State = Still
			e.MoveCooldown = 60
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

func createPlayer(location rl.Vector2) Object {
	return Object{
		Type:               Player,
		Health:             3,
		Speed:              10,
		Damage:             1,
		Position:           location,
		Orientation:        270,
		TurnSpeed:          5,
		Size:               30,
		ShootCooldown:      3,
		ShootTimer:         0,
		InvurnableCooldown: 120,
	}
}

func main() {
	rl.InitWindow(screenWidth, screenHeight, "GeoWars")
	rl.SetExitKey(rl.KeyEscape)

	rl.SetTargetFPS(FPS)

	initObjects()
	player := createPlayer(center)
	allocObj(&player)

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
			if rl.IsKeyDown(rl.KeySpace) {
				if player.ShootTimer > player.ShootCooldown {
					b := Object{
						Type:        Bullet,
						Speed:       5,
						Position:    translate(player.Position, player.Orientation, 25),
						Orientation: player.Orientation,
						Size:        2,
						State:       Moving,
					}
					allocObj(&b)
					player.ShootTimer = 0
				}
			}

			if rl.IsKeyDown(rl.KeyW) {
				moveForward(&player)
			}

			if rl.IsKeyDown(rl.KeyA) {
				rotate(&player, LEFT)
			}

			if rl.IsKeyDown(rl.KeyD) {
				rotate(&player, RIGHT)
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
