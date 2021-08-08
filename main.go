package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Bot parameters
var (
	GuildID = flag.String("guild", "", "if specifide, you can install command only one server")
	AppID   = flag.String("app", "", "Application ID")
)

func init() { flag.Parse() }

func main() {

	s, err := discordgo.New("Bot " + os.Getenv("TOKEN"))
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		// コマンドが実行されたときの振る舞い
		if i.Type == discordgo.InteractionApplicationCommand {
			if i.ApplicationCommandData().Name == "role" {
				createButtons(s, i)
			}
		}

		// ボタンが押されたときじゃない場合は無視する
		if i.Type != discordgo.InteractionMessageComponent {
			return
		}

		// ボタンで渡された内容をパースする
		// TODO: 実装が壊滅的に悪いのでどうにかする
		partsOfCustomID := strings.Split(i.MessageComponentData().CustomID, ":")
		if len(partsOfCustomID) != 4 {
			return
		}
		prefix := partsOfCustomID[0]
		command := partsOfCustomID[1]
		roleID := partsOfCustomID[2]
		userID := partsOfCustomID[3]

		// ボタンで渡された内容が slash-role と関係ない場合は無視する
		if prefix != "slash-role" {
			return
		}

		// ボタンで渡された内容に応じてロールの付与や削除を行う
		var content string
		switch {
		case command == "cancel":
			content = "キャンセルしました"
		case userID != i.Member.User.ID:
			content = "コマンド打った人とボタン押した人が違うよ"
		case command == "add":
			if err := s.GuildMemberRoleAdd(i.GuildID, userID, roleID); err != nil {
				content = "エラーになったよ"
				log.Printf("error: %s", err.Error())
			} else {
				if role, err := s.State.Role(i.GuildID, roleID); err != nil {
					content = "エラーになったよ"
					log.Printf("error: %s", err.Error())
				} else {
					content = fmt.Sprintf("%sさんにロール「%s」を付与したよ", i.Member.User.Username, role.Name)
				}
			}
		case command == "remove":
			if err := s.GuildMemberRoleRemove(i.GuildID, userID, roleID); err != nil {
				content = "エラーになったよ"
				log.Printf("error: %s", err.Error())
			} else {
				if role, err := s.State.Role(i.GuildID, roleID); err != nil {
					content = "エラーになったよ"
					log.Printf("error: %s", err.Error())
				} else {
					content = fmt.Sprintf("%sさんからロール「%s」を削除したよ", i.Member.User.Username, role.Name)
				}
			}
		}

		// 処理結果を表示する
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    content,
				Components: []discordgo.MessageComponent{},
			},
		})
	})

	if _, err := s.ApplicationCommandCreate(*AppID, *GuildID, &discordgo.ApplicationCommand{
		Name:        "role",
		Description: "change role",
	}); err != nil {
		log.Fatalf("Cannot create slash command: %v", err)
	}

	if err := s.Open(); err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")
}

// createButtons create buttons to set roles by Discord interactions
func createButtons(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// Bot自身のロールを取得する
	var myBotRole *discordgo.Role
	if me, err := s.GuildMember(i.GuildID, s.State.User.ID); err != nil {
		panic(err)
	} else {
		for _, id := range me.Roles {
			if myRole, err := s.State.Role(i.GuildID, id); err != nil {
				panic(err)
			} else if myRole.Managed {
				myBotRole = myRole
				break
			}
		}
		if myBotRole == nil {
			panic(err)
		}
	}

	var buttons []discordgo.MessageComponent

	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		panic(err)
	}
	for _, role := range roles {

		// Bot用の Role だったり、 @everyone だったり、 自分より優先度の高い Role だったら無視する
		if role.Managed || role.Position == 0 || role.Position > myBotRole.Position {
			continue
		}

		// そのロールがメンバーにアタッチされているか確認する
		isAttached := false
		for _, id := range i.Member.Roles {
			if id == role.ID {
				isAttached = true
				break
			}
		}

		// ロールをつけたり削除するためのボタンを作る
		var button discordgo.Button
		if isAttached {
			button = discordgo.Button{
				Label:    fmt.Sprintf("Remove '%s'", role.Name),
				Style:    discordgo.DangerButton,
				Disabled: false,
				CustomID: fmt.Sprintf("slash-role:remove:%s:%s", role.ID, i.Member.User.ID),
			}
		} else {
			button = discordgo.Button{
				Label:    fmt.Sprintf("Add '%s'", role.Name),
				Style:    discordgo.SuccessButton,
				Disabled: false,
				CustomID: fmt.Sprintf("slash-role:add:%s:%s", role.ID, i.Member.User.ID),
			}
		}

		buttons = append(buttons, button)

	}

	// 5つ以上のボタンを同じ行に入れることができないので、5つ毎に違う行に入れるようにする
	var actionsRows []discordgo.MessageComponent
	for i := 0; i < (len(buttons)/5)+1; i++ {
		actionsRow := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{},
		}
		for j, b := range buttons {
			if j/5 == i {
				actionsRow.Components = append(actionsRow.Components, b)
			}
		}
		actionsRows = append(actionsRows, actionsRow)
	}

	// キャンセル用のボタンを追加する
	actionsRows = append(actionsRows, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.SecondaryButton,
				Disabled: false,
				CustomID: "slash-role:cancel::",
			},
		},
	})

	// ボタンを表示する
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "アクションを選択してください",
			Components: actionsRows,
		},
	}); err != nil {
		panic(err)
	}

}
