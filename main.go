package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

// Bot parameters
var (
	GuildID = flag.String("guild", "", "Test guild ID")
	AppID   = flag.String("app", "", "Application ID")
)

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + os.Getenv("TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

func main() {

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		roles, err := s.GuildRoles(i.GuildID)
		if err != nil {
			panic(err)
		}
		var crewmateID string
		for _, role := range roles {
			if role.Name == "クルーメイト" {
				crewmateID = role.ID
				break
			}
		}
		log.Println(crewmateID)

		if i.Type == discordgo.InteractionApplicationCommand {
			if i.ApplicationCommandData().Name == "role" {

				includes := false
				for _, id := range i.Member.Roles {
					if id == crewmateID {
						includes = true
						break
					}
				}

				var button discordgo.Button
				if includes {
					button = discordgo.Button{
						Label:    "Remove 'クルーメイト'",
						Style:    discordgo.DangerButton,
						Disabled: false,
						CustomID: "remove_btn",
					}
				} else {
					button = discordgo.Button{
						Label:    "Add 'クルーメイト'",
						Style:    discordgo.SuccessButton,
						Disabled: false,
						CustomID: "add_btn",
					}
				}

				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "test",
						Components: []discordgo.MessageComponent{
							discordgo.ActionsRow{
								Components: []discordgo.MessageComponent{
									button,
								},
							},
						},
					},
				})
				if err != nil {
					panic(err)
				}
			}
			return
		}

		if i.Type != discordgo.InteractionMessageComponent {
			return
		}

		var content string
		switch i.MessageComponentData().CustomID {
		case "add_btn":
			err := s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, crewmateID)
			if err != nil {
				panic(err)
			}
			content = "added role"
		case "remove_btn":
			err := s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, crewmateID)
			if err != nil {
				panic(err)
			}
			content = "removed role"
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    content,
				Components: []discordgo.MessageComponent{},
			},
		})
	})

	_, err := s.ApplicationCommandCreate(*AppID, *GuildID, &discordgo.ApplicationCommand{
		Name:        "role",
		Description: "change role",
	})

	if err != nil {
		log.Fatalf("Cannot create slash command: %v", err)
	}

	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")
}
