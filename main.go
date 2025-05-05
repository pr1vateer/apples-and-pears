package main

import (
	"embed"
	"image/color"
	"image/png"
	"io/fs"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

const (
	ScreenWidth  = 600
	ScreenHeight = 600
	BoardSize    = 3
	CellSize     = ScreenWidth / BoardSize
	// Scale the images to 80% of the cell size, image size is CellSize
	Scale = float64(CellSize) * 0.8 / float64(CellSize)
)

var (
	backgroundColor = color.RGBA{240, 240, 240, 255}
	lineColor       = color.RGBA{50, 50, 50, 255}
	xColor          = color.RGBA{220, 50, 50, 255}
	oColor          = color.RGBA{50, 50, 220, 255}
	textColor       = color.Black
	basicFont       = basicfont.Face7x13
	pearImage       *ebiten.Image
	appleImage      *ebiten.Image

	//go:embed img/*
	embeddedFiles embed.FS
	assets        fs.FS

	// Create a global random number source
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type Cell int

const (
	Empty Cell = iota
	X
	O
)

type GameState int

const (
	Playing GameState = iota
	XWins
	OWins
	Draw
)

type Game struct {
	board       [BoardSize][BoardSize]Cell
	currentTurn Cell
	state       GameState
	aiDelay     int
}

func NewGame() *Game {
	return &Game{
		currentTurn: X,
		state:       Playing,
	}
}

func (g *Game) Update() error {
	if g.state != Playing {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.Reset()
		}
		return nil
	}

	// Player X's turn (human)
	if g.currentTurn == X {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			row := y / CellSize
			col := x / CellSize

			if row >= 0 && row < BoardSize && col >= 0 && col < BoardSize && g.board[row][col] == Empty {
				g.board[row][col] = g.currentTurn

				if g.checkWin() {
					g.state = XWins
				} else if g.checkDraw() {
					g.state = Draw
				} else {
					g.currentTurn = O
					g.aiDelay = 30 // Wait about half a second before AI moves
				}
			}
		}
		// Player O's turn (AI)
	} else {
		if g.aiDelay > 0 {
			g.aiDelay--
			return nil
		}

		// Make AI move
		g.makeAIMove()

		if g.checkWin() {
			g.state = OWins
		} else if g.checkDraw() {
			g.state = Draw
		} else {
			g.currentTurn = X
		}
	}

	return nil
}

func (g *Game) makeAIMove() {
	// Simple AI: first try to win, then choose random empty cell

	// Check if AI can win in the next move
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			if g.board[row][col] == Empty {
				// Try placing O here
				g.board[row][col] = O
				if g.checkWin() {
					// Leave the winning move in place
					return
				}
				// Undo the move
				g.board[row][col] = Empty
			}
		}
	}

	// Try to take the center if it's empty
	if g.board[1][1] == Empty {
		g.board[1][1] = O
		return
	}

	// Take a random empty cell
	var emptyCells [][2]int
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			if g.board[row][col] == Empty {
				emptyCells = append(emptyCells, [2]int{row, col})
			}
		}
	}

	if len(emptyCells) > 0 {
		randomCell := emptyCells[rng.Intn(len(emptyCells))]
		g.board[randomCell[0]][randomCell[1]] = O
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear the screen
	screen.Fill(backgroundColor)

	// Draw the grid
	for i := 1; i < BoardSize; i++ {
		// Horizontal lines
		vector.StrokeLine(screen, 0, float32(i*CellSize), float32(ScreenWidth), float32(i*CellSize), 1, lineColor, false)
		// Vertical lines
		vector.StrokeLine(screen, float32(i*CellSize), 0, float32(i*CellSize), float32(ScreenHeight), 1, lineColor, false)
	}

	// Draw apples and pears
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			x := float64(col * CellSize)
			y := float64(row * CellSize)

			switch g.board[row][col] {
			case X:
				if appleImage != nil {
					op := &ebiten.DrawImageOptions{}
					// Both should be equal to CellSize
					imgWidth, imgHeight := appleImage.Bounds().Dx(), appleImage.Bounds().Dy()
					op.GeoM.Scale(Scale, Scale)
					op.GeoM.Translate(
						x+(CellSize-float64(imgWidth)*Scale)/2,
						y+(CellSize-float64(imgHeight)*Scale)/2,
					)
					screen.DrawImage(appleImage, op)
				}
			case O:
				if pearImage != nil {
					op := &ebiten.DrawImageOptions{}
					imgWidth, imgHeight := pearImage.Bounds().Dx(), pearImage.Bounds().Dy()
					op.GeoM.Scale(Scale, Scale)
					op.GeoM.Translate(
						x+(CellSize-float64(imgWidth)*Scale)/2,
						y+(CellSize-float64(imgHeight)*Scale)/2,
					)
					screen.DrawImage(pearImage, op)
				}
			}
		}
	}

	switch g.state {
	case Playing:
		turn := "Apple's turn"
		if g.currentTurn == O {
			turn = "Pear's turn"
		}
		text.Draw(screen, turn, basicFont, 10, ScreenHeight-10, textColor)
	case XWins:
		text.Draw(screen, "Apple wins! Click to play again", basicFont, 10, ScreenHeight-10, textColor)
	case OWins:
		text.Draw(screen, "Pear wins! Click to play again", basicFont, 10, ScreenHeight-10, textColor)
	case Draw:
		text.Draw(screen, "Draw! Click to play again", basicFont, 10, ScreenHeight-10, textColor)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func (g *Game) Reset() {
	for i := range g.board {
		for j := range g.board[i] {
			g.board[i][j] = Empty
		}
	}
	g.currentTurn = X
	g.state = Playing
	g.aiDelay = 0
}

func (g *Game) checkWin() bool {
	// Check rows
	for row := 0; row < BoardSize; row++ {
		if g.board[row][0] != Empty && g.board[row][0] == g.board[row][1] && g.board[row][1] == g.board[row][2] {
			return true
		}
	}

	// Check columns
	for col := 0; col < BoardSize; col++ {
		if g.board[0][col] != Empty && g.board[0][col] == g.board[1][col] && g.board[1][col] == g.board[2][col] {
			return true
		}
	}

	// Check diagonals
	if g.board[0][0] != Empty && g.board[0][0] == g.board[1][1] && g.board[1][1] == g.board[2][2] {
		return true
	}
	if g.board[0][2] != Empty && g.board[0][2] == g.board[1][1] && g.board[1][1] == g.board[2][0] {
		return true
	}

	return false
}

func (g *Game) checkDraw() bool {
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			if g.board[row][col] == Empty {
				return false
			}
		}
	}
	return true
}

func loadImage(name string) *ebiten.Image {
	var f, err = assets.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	eImg := ebiten.NewImageFromImage(img)

	return eImg
}

func main() {
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Apples and Pears")

	var err error
	assets, err = fs.Sub(embeddedFiles, "img")
	if err != nil {
		panic(err)
	}

	pearImage = loadImage("pear.png")

	appleImage = loadImage("apple.png")

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
