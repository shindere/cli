package command

import (
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
	"github.com/cli/cli/internal/run"
	"github.com/cli/cli/utils"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

type Geometry struct {
	TermWidth  int
	TermHeight int
	Density    float64
}

type Player struct {
	X    int
	Y    int
	Char string
}

type Commit struct {
	Email string
	Sha   string
	Char  string
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
	// TODO color
	// TODO character mapping
	// TODO respect multiple author commits
	// TODO static version

	// starting from top left, print a character per commit. do it as one line so it wraps.

	//ctx := contextForCommand(cmd)
	//client, err := apiClientForContext(ctx)
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
			// TODO dedupe chosen characters
			//char := emailToChar(client, email)
			char := emailToChar(email)
			charUser[char] = email
			userChar[email] = char
		}
		char := userChar[email]

		colorFunc := shaToColorFunc(sha)
		colorChar := fmt.Sprintf("%s", colorFunc(char))
		commits = append(commits, &Commit{email, sha, colorChar})
	}

	termWidth, termHeight, err := terminal.GetSize(int(outFile.Fd()))
	if err != nil {
		return err
	}

	termWidth -= 10
	termHeight -= 10

	seed := computeSeed("TODO REPO NAME OR SOMETHING")
	rand.Seed(seed)

	geo := &Geometry{
		TermWidth:  termWidth,
		TermHeight: termHeight,
		// TODO based on number of commits/cells instead of just hardcoding
		Density: 0.4,
	}

	//cellCount := float64(termWidth * termHeight)
	//flowerCount := float64(len(flowers))

	//density := (cellCount / flowerCount)
	//fmt.Println("DENSITY", density)

	player := &Player{0, 0, utils.Bold("@")}

	clear()
	drawGarden(out, commits, player, geo)

	// thanks stackoverflow https://stackoverflow.com/a/17278776
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	var b []byte = make([]byte, 1)
	for {
		os.Stdin.Read(b)
		break
	}

	fmt.Println()

	return nil
}

func drawGarden(out io.Writer, commits []*Commit, player *Player, geo *Geometry) {
	statusLine := ""

	// TODO intelligent density. for now just every-other
	// TODO variety of grass characters
	// TODO animate wind blowing on the grass
	gardenRows := []string{}
	cellIx := 0
	grassChar := ","
	for y := 0; y < geo.TermHeight; y++ {
		row := ""
		for x := 0; x < geo.TermWidth; x++ {
			underPlayer := (player.X == x && player.Y == y)
			char := ""

			if cellIx == len(commits)-1 {
				char = utils.Green(grassChar)
				if underPlayer {
					char = player.Char
					statusLine = "You're standing on a patch of grass in a field of wildflowers."
				}
			} else {
				chance := rand.Float64()
				if chance <= geo.Density {
					commit := commits[cellIx]
					char = commit.Char
					if underPlayer {
						char = player.Char
						statusLine = fmt.Sprintf("You're standing at a flower called %s planted by %s.", commit.Sha, commit.Email)
					}

				} else {
					char = utils.Green(grassChar)
					if underPlayer {
						char = player.Char
						statusLine = "You're standing on a patch of grass in a field of wildflowers."
					}
				}
				cellIx++
			}

			row += char
		}
		gardenRows = append(gardenRows, row)
	}

	for _, r := range gardenRows {
		fmt.Fprintln(out, r)
	}
	fmt.Fprintf(out, utils.Bold(statusLine))
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

		//fmt.Println(sha[0:2], sha[2:4], sha[4:6])
		//fmt.Println("COLOR CODE:", sha, red, green, blue)

		// TODO figure out why escaping not working
		return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", red, green, blue, c)
	}
}

//func emailToChar(client *api.Client, email string) string {
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
	//type item struct {
	//	Login string
	//}
	//var response struct {
	//	Items []item
	//}

	//err := client.REST("GET", fmt.Sprintf("search/users?q=%s+in:email", email), nil, &response)

	//if err != nil {
	//	// TODO
	//	fmt.Fprintf(os.Stderr, "failed to use search api: %w\n", err)
	//}

	//fmt.Printf("%#v\n", response)
}

func outputLines(output []byte) []string {
	lines := strings.TrimSuffix(string(output), "\n")
	return strings.Split(lines, "\n")
}

func computeSeed(seed string) int64 {
	return 1234567890
}
