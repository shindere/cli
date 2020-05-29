package command

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/cli/git"
	"github.com/cli/cli/internal/ghrepo"
	"github.com/cli/cli/internal/run"
	"github.com/cli/cli/utils"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

type Geometry struct {
	Width      int
	Height     int
	Density    float64
	Repository ghrepo.Interface
}

type Player struct {
	X    int
	Y    int
	Char string
	Geo  *Geometry
}

type Commit struct {
	Email  string
	Handle string
	Sha    string
	Char   string
}

type Cell struct {
	Char       string
	StatusLine string
}

const (
	DirUp = iota
	DirDown
	DirLeft
	DirRight
)

type Direction = int

func (p *Player) move(direction Direction) {
	switch direction {
	case DirUp:
		if p.Y == 0 {
			return
		}
		p.Y--
	case DirDown:
		if p.Y == p.Geo.Height {
			return
		}
		p.Y++
	case DirLeft:
		if p.X == 0 {
			return
		}
		p.X--
	case DirRight:
		if p.X == p.Geo.Width {
			return
		}
		p.X++
	}

	return
}

func init() {
	repoCmd.AddCommand(repoGardenCmd)
}

var repoGardenCmd = &cobra.Command{
	Use:   "garden",
	Short: "A unique piece of art derived from git history",
	RunE:  repoGarden,
}

func repoGarden(cmd *cobra.Command, args []string) error {
	// TODO respect multiple author commits
	// TODO static version
	// TODO get contributor list
	// TODO switch to GH usernames
	// TODO better bounds handling
	// TODO repo seed
	// TODO add a stream

	ctx := contextForCommand(cmd)
	client, err := apiClientForContext(ctx)
	if err != nil {
		return err
	}
	baseRepo, err := determineBaseRepo(client, cmd, ctx)
	if err != nil {
		return err
	}
	//repo, err := api.GitHubRepo(client, baseRepo)
	//if err != nil {
	//	return err
	//}

	out := colorableOut(cmd)

	isTTY := false
	outFile, isFile := out.(*os.File)
	if isFile {
		isTTY = utils.IsTerminal(outFile)
		if isTTY {
			// FIXME: duplicates colorableOut
			out = utils.NewColorable(outFile)
		}
	}

	if !isTTY {
		return errors.New("TODO")
	}

	commitsCmd := git.GitCommand("log", "--pretty=format:%h,%ae")
	output, err := run.PrepareCmd(commitsCmd).Output()
	if err != nil {
		return err
	}

	commitLines := outputLines(output)

	userChar := map[string]string{}
	charUser := map[string]string{}

	commits := []*Commit{}

	for _, line := range commitLines {
		parts := strings.Split(line, ",")
		sha := parts[0]
		email := parts[1]

		if _, ok := userChar[email]; !ok {
			char := emailToChar(email)
			charUser[char] = email
			userChar[email] = char
		}
		char := userChar[email]

		colorFunc := shaToColorFunc(sha)
		colorChar := fmt.Sprintf("%s", colorFunc(char))
		parts = strings.Split(email, "@")
		handle := parts[0]
		commits = append(commits, &Commit{email, handle, sha, colorChar})
	}

	termWidth, termHeight, err := terminal.GetSize(int(outFile.Fd()))
	if err != nil {
		return err
	}

	termWidth -= 10
	termHeight -= 10

	seed := computeSeed(ghrepo.FullName(baseRepo))
	rand.Seed(seed)

	geo := &Geometry{
		Width:      termWidth,
		Height:     termHeight,
		Repository: baseRepo,
		// TODO based on number of commits/cells instead of just hardcoding
		Density: 0.3,
	}

	player := &Player{0, 0, utils.Bold("@"), geo}

	clear()
	garden := plantGarden(commits, geo)
	drawGarden(out, garden, player)

	// thanks stackoverflow https://stackoverflow.com/a/17278776
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	var b []byte = make([]byte, 1)
	for {
		os.Stdin.Read(b)

		quitting := false
		switch {
		case isLeft(b):
			player.move(DirLeft)
		case isRight(b):
			player.move(DirRight)
		case isUp(b):
			player.move(DirUp)
		case isDown(b):
			player.move(DirDown)
		case isQuit(b):
			quitting = true
		}

		if quitting {
			break
		}

		clear()
		drawGarden(out, garden, player)
	}

	fmt.Println()
	fmt.Println(utils.Bold("You turn and walk away from the wildflower garden..."))

	return nil
}

// TODO fix arrow keys

func isLeft(b []byte) bool {
	return bytes.EqualFold(b, []byte("a")) || bytes.EqualFold(b, []byte("h"))
}

func isRight(b []byte) bool {
	return bytes.EqualFold(b, []byte("d")) || bytes.EqualFold(b, []byte("l"))
}

func isDown(b []byte) bool {
	return bytes.EqualFold(b, []byte("s")) || bytes.EqualFold(b, []byte("j"))
}

func isUp(b []byte) bool {
	return bytes.EqualFold(b, []byte("w")) || bytes.EqualFold(b, []byte("k"))
}

func isQuit(b []byte) bool {
	return bytes.EqualFold(b, []byte("q"))
}

func plantGarden(commits []*Commit, geo *Geometry) [][]*Cell {
	cellIx := 0
	grassCell := &Cell{utils.Green(","), "You're standing on a patch of grass in a field of wildflowers."}
	garden := [][]*Cell{}
	for y := 0; y < geo.Height; y++ {
		if cellIx == len(commits)-1 {
			break
		}
		garden = append(garden, []*Cell{})
		for x := 0; x < geo.Width; x++ {
			if y == 0 && (x < geo.Width/2 || x > geo.Width/2) {
				garden[y] = append(garden[y], &Cell{
					Char:       " ",
					StatusLine: "You're standing by a wildflower garden. There is a light breeze.",
				})
				continue
			} else if y == 0 && x == geo.Width/2 {
				garden[y] = append(garden[y], &Cell{
					Char:       utils.RGB(139, 69, 19, "+"),
					StatusLine: fmt.Sprintf("You're standing in front of a weather-beaten sign that says %s.", ghrepo.FullName(geo.Repository)),
				})
				continue
			}

			if cellIx == len(commits)-1 {
				garden[y] = append(garden[y], grassCell)
				continue
			}

			chance := rand.Float64()
			if chance <= geo.Density {
				commit := commits[cellIx]
				garden[y] = append(garden[y], &Cell{
					Char:       commits[cellIx].Char,
					StatusLine: fmt.Sprintf("You're standing at a flower called %s planted by %s.", commit.Sha, commit.Handle),
				})
				cellIx++
			} else {
				garden[y] = append(garden[y], grassCell)
			}
		}
	}

	return garden
}

func drawGarden(out io.Writer, garden [][]*Cell, player *Player) {
	statusLine := ""
	for y, gardenRow := range garden {
		for x, gardenCell := range gardenRow {
			char := ""
			underPlayer := (player.X == x && player.Y == y)
			if underPlayer {
				statusLine = gardenCell.StatusLine
				char = utils.Bold(player.Char)
			} else {
				char = gardenCell.Char
			}

			fmt.Fprint(out, char)
		}
		fmt.Fprintln(out)
	}

	fmt.Println()
	fmt.Fprintln(out, utils.Bold(statusLine))
}

func shaToColorFunc(sha string) func(string) string {
	return func(c string) string {
		red, err := strconv.ParseInt(sha[0:2], 16, 64)
		if err != nil {
			panic(err)
		}

		green, err := strconv.ParseInt(sha[2:4], 16, 64)
		if err != nil {
			panic(err)
		}

		blue, err := strconv.ParseInt(sha[4:6], 16, 64)
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", red, green, blue, c)
	}
}

func emailToChar(email string) string {
	numRE := regexp.MustCompile(`^[0-9]+$`)
	parts := strings.Split(email, "@")
	handle := parts[0]
	if strings.Contains(handle, "+") {
		parts = strings.Split(handle, "+")
		if numRE.MatchString(parts[0]) {
			return string(parts[1][0])
		} else {
			return string(parts[0][0])
		}
	} else {
		return string(handle[0])
	}
}

func outputLines(output []byte) []string {
	lines := strings.TrimSuffix(string(output), "\n")
	return strings.Split(lines, "\n")
}

func computeSeed(seed string) int64 {
	runes := []rune(seed)

	lol := ""

	for _, r := range runes {
		lol += fmt.Sprintf("%d", int(r))
	}

	result, err := strconv.ParseInt(lol, 10, 64)
	if err != nil {
		panic(err)
	}

	return result
}
