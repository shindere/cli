package command

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/cli/git"
	"github.com/cli/cli/internal/run"
	"github.com/spf13/cobra"
)

func init() {
	repoCmd.AddCommand(repoQuiltCmd)
}

var repoQuiltCmd = &cobra.Command{
	Use:   "quilt",
	Short: "A unique piece of art derived from git history",
	RunE:  repoQuilt,
}

func repoQuilt(cmd *cobra.Command, args []string) error {
	// TODO color
	// TODO character mapping
	// TODO respect multiple author commits

	// starting from top left, print a character per commit. do it as one line so it wraps.

	//ctx := contextForCommand(cmd)
	//client, err := apiClientForContext(ctx)
	//if err != nil {
	//	return err
	//}

	commitsCmd := git.GitCommand("log", "--pretty=format:%h,%ae")
	output, err := run.PrepareCmd(commitsCmd).Output()
	if err != nil {
		return err
	}

	commitLines := outputLines(output)

	userChar := map[string]string{}
	charUser := map[string]string{}

	out := colorableOut(cmd)

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
		fmt.Fprintf(out, "%s", colorFunc(char))
	}

	fmt.Println()

	return nil
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
