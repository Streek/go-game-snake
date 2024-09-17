package main

import (
    "fmt"
    "math/rand"
    "os"
    "os/signal"
    "strings"
    "sync"
    "syscall"
    "time"

    "golang.org/x/term"
)

type Point struct {
    X int
    Y int
}

type Game struct {
    Width       int
    Height      int
    Snake       []Point
    Direction   string
    Fruit       Point
    Score       int
    GameOver    bool
    Mutex       sync.Mutex
    InputBuffer chan rune
}

func (g *Game) Initialize() {
    g.Width = 60 // Adjusted to fit terminal width
    g.Height = 40
    g.Direction = "RIGHT"
    g.Snake = []Point{
        {X: g.Width / 2, Y: g.Height / 2},
    }
    g.SpawnFruit()
    g.Score = 0
    g.GameOver = false
    g.InputBuffer = make(chan rune, 1)
}

func (g *Game) SpawnFruit() {
    for {
        fruit := Point{
            X: rand.Intn(g.Width),
            Y: rand.Intn(g.Height),
        }
        overlaps := false
        for _, s := range g.Snake {
            if s.X == fruit.X && s.Y == fruit.Y {
                overlaps = true
                break
            }
        }
        if !overlaps {
            g.Fruit = fruit
            break
        }
    }
}

func (g *Game) Update() {
    g.Mutex.Lock()
    defer g.Mutex.Unlock()

    head := g.Snake[0]
    var newHead Point

    switch g.Direction {
    case "UP":
        newHead = Point{X: head.X, Y: head.Y - 1}
    case "DOWN":
        newHead = Point{X: head.X, Y: head.Y + 1}
    case "LEFT":
        newHead = Point{X: head.X - 1, Y: head.Y}
    case "RIGHT":
        newHead = Point{X: head.X + 1, Y: head.Y}
    }

    if newHead.X < 0 || newHead.X >= g.Width || newHead.Y < 0 || newHead.Y >= g.Height {
        g.GameOver = true
        return
    }

    for _, s := range g.Snake {
        if s.X == newHead.X && s.Y == newHead.Y {
            g.GameOver = true
            return
        }
    }

    g.Snake = append([]Point{newHead}, g.Snake...)

    if newHead.X == g.Fruit.X && newHead.Y == g.Fruit.Y {
        g.Score++
        g.SpawnFruit()
    } else {
        g.Snake = g.Snake[:len(g.Snake)-1]
    }
}

func (g *Game) Draw() {
    g.Mutex.Lock()
    defer g.Mutex.Unlock()

    fmt.Print("\033[2J\033[H")

    var output strings.Builder

    output.WriteString("\r┌")
    for i := 0; i < g.Width; i++ {
        output.WriteString("──")
    }
    output.WriteString("┐\n")

    for y := 0; y < g.Height; y++ {
        output.WriteString("\r│")
        for x := 0; x < g.Width; x++ {
            cell := "  "
            occupied := false

            if g.Fruit.X == x && g.Fruit.Y == y {
                cell = "🍎"
                occupied = true
            }

            if !occupied {
                for i, s := range g.Snake {
                    if s.X == x && s.Y == y {
                        if i == 0 {
                            cell = "🟢" // Head
                        } else {
                            cell = "🟩" // Body
                        }
                        break
                    }
                }
            }

            output.WriteString(cell)
        }
        output.WriteString("│\n\r")
    }

    output.WriteString("\r└")
    for i := 0; i < g.Width; i++ {
        output.WriteString("──")
    }
    output.WriteString("┘\n\r")

    output.WriteString(fmt.Sprintf("\rScore: %d\n\r", g.Score))

    fmt.Print(output.String())
}

func (g *Game) HandleInput() {
	for input := range g.InputBuffer {
			g.Mutex.Lock()
			switch input {
			case 'w', 'W', '↑':
					if g.Direction != "DOWN" {
							g.Direction = "UP"
					}
			case 's', 'S', '↓':
					if g.Direction != "UP" {
							g.Direction = "DOWN"
					}
			case 'a', 'A', '←':
					if g.Direction != "RIGHT" {
							g.Direction = "LEFT"
					}
			case 'd', 'D', '→':
					if g.Direction != "LEFT" {
							g.Direction = "RIGHT"
					}
			case 'q', 'Q':
					g.GameOver = true
			}
			g.Mutex.Unlock()
	}
}

func (g *Game) ReadInput() {
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
			panic(err)
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	buf := make([]byte, 3) // Buffer to hold up to 3 bytes
	for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
					continue
			}
			if buf[0] == '\x1b' {
					if n == 1 {
							nExtra, err := os.Stdin.Read(buf[1:])
							if err != nil || nExtra != 2 {
									continue
							}
					}
					if buf[1] == '[' {
							switch buf[2] {
							case 'A': // Up arrow
									g.InputBuffer <- '↑'
							case 'B': // Down arrow
									g.InputBuffer <- '↓'
							case 'C': // Right arrow
									g.InputBuffer <- '→'
							case 'D': // Left arrow
									g.InputBuffer <- '←'
							}
					}
					buf = make([]byte, 3)
			} else {
					g.InputBuffer <- rune(buf[0])
			}
	}
}

func main() {
    rand.Seed(time.Now().UnixNano())

    game := Game{}
    game.Initialize()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    go func() {
        <-sigChan
        game.GameOver = true
    }()

    go game.ReadInput()
    go game.HandleInput()

    ticker := time.NewTicker(time.Second / 10)
    defer ticker.Stop()

    for !game.GameOver {
        select {
        case <-ticker.C:
            game.Update()
            game.Draw()
        }
    }

    // Game over screen
    fmt.Printf("Final Score: %d\n\r", game.Score)
}
