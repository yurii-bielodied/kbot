package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	telebot "gopkg.in/telebot.v4"
)

var (
	// Teletoken bot
	TeleToken = os.Getenv("TELE_TOKEN")
)

// kbotCmd represents the kbot command
var kbotCmd = &cobra.Command{
	Use:     "kbot",
	Aliases: []string{"start"},
	Short:   "Telegram bot",
	Long: `A Telegram bot.
The bot accepts some commands.

Usage:
  hello     - Get a greeting from the bot
  time      - Get system time`,
	Run: func(cmd *cobra.Command, args []string) {
		if TeleToken == "" {
			log.Fatal("TELE_TOKEN environment variable is not set")
		}

		fmt.Printf("kbot %s started\n", appVersion)

		kbot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  TeleToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			log.Fatalf("Please check TELE_TOKEN env variable. %s", err)
		}

		kbot.Handle(telebot.OnText, func(m telebot.Context) error {
			log.Printf("Received message: %s", m.Text())
			payload := m.Message().Payload

			switch payload {
			case "hello":
				return m.Send(fmt.Sprintf("Hello I'm Kbot %s!", appVersion))

			case "time":
				return m.Send(fmt.Sprintf("Current time is %v", time.Now().Format("2006-01-02 15:04:05")))

			default:
				return m.Send("Hello from Kbot!")
			}
		})

		kbot.Start()
	},
}

func init() {
	rootCmd.AddCommand(kbotCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kbotCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kbotCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// func main() {
// 	pref := telebot.Settings{
// 		Token:  os.Getenv("TOKEN"),
// 		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
// 	}

// 	b, err := telebot.NewBot(pref)
// 	if err != nil {
// 		log.Fatal(err)
// 		return
// 	}

// 	b.Handle("/hello", func(c telebot.Context) error {
// 		return c.Send("Hello!")
// 	})

// 	b.Start()
// }
