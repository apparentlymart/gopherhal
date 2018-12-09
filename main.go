package main

import (
	"fmt"
	"os"

	"github.com/apparentlymart/gopherhal/ghal"
	prompt "github.com/c-bata/go-prompt"
	"github.com/spf13/pflag"
)

func main() {
	brainFile := pflag.String("brain", "gopherhal.brain", "file to use to load/save the bot's brain")
	debug := pflag.Bool("debug", false, "show verbose word tagging during chat")
	pflag.Parse()
	args := pflag.Args()
	if len(args) == 0 {
		errUsage()
	}

	switch args[0] {
	case "chat":
		if len(args) != 1 {
			errUsage()
		}
		os.Exit(chat(*brainFile, *debug))
	case "train":
		os.Exit(train(*brainFile, args[1:]))
	default:
		errUsage()
	}
}

func chat(brainFile string, debug bool) int {
	brain, err := ghal.LoadBrainFile(brainFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading brain from %q: %s\n", brainFile, err)
		return 1
	}
	for {
		inp := prompt.Input("> ", noComplete)
		if inp == "exit" || inp == "quit" {
			fmt.Printf("bye!\n")
			break
		}
		sentences, err := ghal.ParseText(inp)
		if err != nil {
			fmt.Printf("sorry... i'm afraid I can't make any sense of that :(\n%s\n", err)
			continue
		}
		if debug {
			fmt.Printf("Here's how I understood your message:\n")
			for _, sentence := range sentences {
				fmt.Printf("- %s\n", sentence.StringTagged())
			}
			fmt.Printf("\n")
		}
		reply := brain.MakeReply(sentences...)
		if len(reply) == 0 {
			reply = brain.MakeQuestion()
		}
		if len(reply) == 0 {
			fmt.Printf("i am speechless :(\n")
			continue
		}
		reply = reply.TrimPeriod()
		if debug {
			fmt.Printf("My response:\n- %s\n", reply.StringTagged())
		} else {
			fmt.Printf("%s\n", reply)
		}

		// Learn the sentences the user typed, but we'll trim off trailing
		// periods to preserve the bot's conversational style.
		for _, sentence := range sentences {
			brain.AddSentence(sentence.TrimPeriod())
		}
	}
	safeSaveBrain(brain, brainFile)
	return 0
}

func train(brainFile string, corpusFiles []string) int {
	return 0
}

func errUsage() {
	os.Stderr.WriteString("Usage: gopherhal <chat|train>\n")
	os.Exit(1)
}

func noComplete(d prompt.Document) []prompt.Suggest {
	return nil
}

func safeSaveBrain(brain *ghal.Brain, filename string) {
	tempName := "." + filename + ".new"
	err := brain.SaveFile(tempName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save brain: %s\n", err)
		os.Exit(1)
	}
	err = os.Rename(tempName, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to replace brain snapshot with new snapshot: %s\n", err)
		os.Exit(1)
	}
}