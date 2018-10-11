package selpg

import(
	"io"
	"os"
	"fmt"
	"bufio"
	"log"
	"os/exec"
	"bytes"
	"github.com/spf13/pflag"
)

type Selpg struct {
	startPage, endPage, pageLines, totalPages int
	destination, inputFile string
	fromFeed bool
}

func NewSelpg(start, end, lines int, dest, input string, useFormFeed bool) Selpg {
	return Selpg{
		start,
		end,
		lines,
		end - start + 1,
		dest,
		input,
		useFormFeed,
	}
}

func (sp Selpg) GetPages(writer io.Writer) (err error) {
	var reader io.Reader
	if sp.inputFile != "" {
		file, err := os.Open(sp.inputFile)
		if err != nil {
			fmt.Printf("%v: Could not open file: %v, %v\n", os.Args[0], sp.inputFile, err)
			os.Exit(1)
		}
		reader = file
	} 
	else {
		reader = os.Stdin
	}
	bufReader := bufio.NewReader(reader)
	bufWriter := bufio.NewWriter(writer)
	var totalPages int

	if sp.fromFeed {
		currentPage := 1
		for runningFlag := true; runningFlag ;{
			ch, err := bufReader.ReadByte()
			if err != nil {
				if err == io.EOF {
					runningFlag = false
				} 
				else {
					log.Fatalf("Error: %v\n", err)
					os.Exit(1)
				}
			}
			if currentPage >= sp.startPage && currentPage <= sp.endPage {
				bufWriter.WriteByte(ch)
			}
			if ch == '\f' {
				bufWriter.Flush()
				currentPage += 1
			}
		}
		totalPages = currentPage
	} 
	else {
		currentLine := 1
		for runningFlag := true; runningFlag ;{
			line, err := bufReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					runningFlag = false
				} else {
					log.Fatalf("Error: %v\n", err)
					os.Exit(1)
				}
			}
			if currentLine >= sp.pageLines*(sp.startPage-1)+1 &&
				currentLine <= sp.totalPages * sp.pageLines {
				bufWriter.WriteString(line)
			}
			bufWriter.Flush()
			currentLine += 1
		}
		totalPages = currentLine / sp.pageLines + 1
	}
	if totalPages < sp.startPage {
		err = errors.New(fmt.Sprintf("start_page (%v) greater than total pages (%v), no output written", sp.startPage, totalPages))
	} else if totalPages < sp.endPage {
		err = errors.New(fmt.Sprintf("end_page (%v) greater than total pages (%v), less output than expected", sp.endPage, totalPages))
	}
	bufWriter.Flush()
	return err
}

func (sp Selpg) Run() {
	var writer io.Writer

	var cmd *exec.Cmd
	piper, pipew := io.Pipe()
	buf := new(bytes.Buffer)
	if sp.destination != "" {
		cmd = exec.Command("lp", fmt.Sprintf("-d%v", sp.destination))

		writer = buf

		stderr, _ := cmd.StderrPipe()
		go func() {
			defer pipew.Close()
			io.Copy(pipew, stderr)
		}()

	} else {
		writer = os.Stdout
	}
	err := sp.GetPages(writer)

	if sp.destination != "" {
		cmd.Stdin = buf
		cmd.Run()
		io.Copy(os.Stderr, piper)
		defer bufio.NewWriter(os.Stderr).Flush()
		cmd.Wait()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %v\n", os.Args[0], err)
	}

	fmt.Fprintf(os.Stderr, "%v: Done\n", os.Args[0])
}