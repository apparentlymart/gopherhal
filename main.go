package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/apparentlymart/gopherhal/ghal"
	"github.com/apparentlymart/gopherhal/trainhal"
	prompt "github.com/c-bata/go-prompt"
	"github.com/spf13/pflag"
)

var why = ghal.MakeWord("WRB", "why")
var because = ghal.MakeWord("IN", "because")

func main() {
	brainFile := pflag.String("brain", "gopherhal.brain", "file to use to load/save the bot's brain")
	debug := pflag.Bool("debug", false, "show verbose word tagging during chat")
	pflag.Parse()
	args := pflag.Args()
	if len(args) == 0 {
		errUsage()
	}

	if *debug {
		ghal.SetDebugLog(os.Stderr, "brain: ")
	}
	rand.Seed(time.Now().Unix())

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

	// We'll open with a question, to start the "discussion".
	opener := brain.MakeQuestion()
	if len(opener) > 0 {
		fmt.Printf("hello! %s\n", opener)
	} else {
		fmt.Printf("hello!\n")
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

		var reply ghal.Sentence

		// If this seems to be a "why" question then we'll try to randomly
		// select a "because..." sentence to respond with.
		if len(sentences) > 0 && len(sentences[0]) > 0 {
			if sentences[0][0] == why {
				reply = brain.MakeReason()
			}
		}

		if len(reply) == 0 {
			reply = brain.MakeReply(sentences...)
		}
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
	if len(corpusFiles) == 0 {
		os.Stderr.WriteString("Usage: gopherhal train <corpus-file>...\n")
		return 1
	}

	brain, err := ghal.LoadBrainFile(brainFile)
	if os.IsNotExist(err) {
		log.Printf("Starting training with a new, empty brain")
		brain = ghal.NewBrain()
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading brain from %q: %s\n", brainFile, err)
		return 1
	}

	for _, filename := range corpusFiles {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open %s: %s\n", filename, err)
			return 1
		}

		log.Printf("Reading training content from %s...", filename)
		log.Print("Content extraction can be slow, so larger files may take minutes to import.")
		sentences, err := trainhal.ParseTrainingInput(f, filename, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read %s: %s\n", filename, err)
			return 1
		}

		log.Printf("Sentences found: %d", len(sentences))
		for i, sentence := range sentences {
			if i == 5 {
				log.Printf("- (etc...)")
				break
			}
			log.Printf("- %s", sentence)
		}
		brain.AddSentences(sentences)

		// Overwrite our initial brain file after each successful import.
		safeSaveBrain(brain, brainFile)
	}

	log.Printf("All done! Update brain saved in %s", brainFile)

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
