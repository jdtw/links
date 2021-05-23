package keybase

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jdtw/links/pkg/client"
	"github.com/keybase/go-keybase-chat-bot/kbchat"
)

type linksClient struct {
	c *client.Client
}

func ChatBot(keybaseLoc string, links *client.Client) error {
	kbc, err := kbchat.Start(kbchat.RunOptions{KeybaseLocation: keybaseLoc})
	if err != nil {
		return err
	}
	me := kbc.GetUsername()
	log.Printf("Started chat as user %q", me)

	lc := &linksClient{links}

	sub, err := kbc.ListenForNewTextMessages()
	if err != nil {
		return err
	}

	for {
		m, err := sub.Read()
		if err != nil {
			log.Printf("read message failed: %v", err)
			continue
		}
		if m.Message.Sender.Username == me || m.Message.Content.TypeName != "text" {
			continue
		}

		cmd := strings.Fields(m.Message.Content.Text.Body)
		log.Print(cmd)
		if len(cmd) == 0 {
			// TODO(jdtw): Send usage...
			kbc.SendMessage(m.Message.Channel, "Bad request!")
			continue
		}

		var reply string
		action := strings.ToLower(cmd[0])
		switch action {
		case "add":
			reply, err = lc.add(cmd[1:]...)
		case "list":
			reply, err = lc.list(cmd[1:]...)
		case "rm":
			reply, err = lc.rm(cmd[1:]...)
		default:
			reply = fmt.Sprintf("Unknown command: %q", action)
		}
		if err != nil {
			if _, err := kbc.SendMessage(m.Message.Channel, "%s failed: %v", action); err != nil {
				log.Printf("kbc.SendMessage failed: %v", err)
			}
			continue
		}
		if reply != "" {
			if _, err := kbc.SendMessage(m.Message.Channel, reply); err != nil {
				log.Printf("kbc.SendMessage failed: %v", err)
			}
		}
	}
}

func (lc *linksClient) add(args ...string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("got %d args, expected 2", len(args))
	}
	if err := lc.c.Put(args[0], args[1]); err != nil {
		return "", err
	}
	return fmt.Sprintf("Added %s!", args[0]), nil
}

func (lc *linksClient) list(args ...string) (string, error) {
	m, err := lc.c.List()
	if err != nil {
		return "", err
	}
	ls := make([]string, 0, len(m))
	for k, v := range m {
		ls = append(ls, fmt.Sprintf("%s %s", k, v))
	}
	sort.Strings(ls)
	return strings.Join(ls, "\n"), nil
}

func (lc *linksClient) rm(args ...string) (string, error) {
	for _, a := range args {
		if err := lc.c.Delete(a); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("Deleted %s!", strings.Join(args, ", ")), nil
}
