package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	redis "github.com/redis/go-redis/v9"
)

type Config struct {
	RedisURL string `json:"redis_url"`
}

func main() {
	redisURL := flag.String("redis_url", "redis://localhost/", "Redis URL")
	flag.Parse()
	redisOptions, err := redis.ParseURL(*redisURL)
	if err != nil {
		log.Fatalf("Invalid Redis URL: %v", err)
	}

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer sigCancel()

	rdb := redis.NewClient(redisOptions)

	cmdSub := rdb.PSubscribe(sigCtx, "activesync.command.*")
	defer cmdSub.Close()

	_, err = cmdSub.Receive(sigCtx)
	if err != nil {
		log.Fatalf("Failed to subscribe to commands: %v", err)
	}

	log.Println("Redis connected. Watching for commands")

	handleCommands(sigCtx, cmdSub)
}

func handleCommands(ctx context.Context, cmdSub *redis.PubSub) {
	ch := cmdSub.Channel()
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down command handler")
			return
		case msg := <-ch:
			// Get the last part of the subject to determine the command
			subjectParts := strings.Split(msg.Channel, ".")
			command := subjectParts[len(subjectParts)-1]

			// Split the payload into "<username> <deviceid>"
			payloadParts := strings.SplitN(msg.Payload, " ", 2)
			username := payloadParts[0]
			// some commands accept user only, so deviceID is optional
			deviceID := ""
			if len(payloadParts) > 1 {
				deviceID = payloadParts[1]
			}

			// Check if user is handled by local
			if isLocal, err := isLocalUser(ctx, username); err != nil {
				log.Printf("Error checking if user is local: %v", err)
				continue
			} else if !isLocal {
				log.Printf("User '%s' is not a local user. Skipping command '%s'", username, command)
				continue
			}

			var err error
			var output string
			switch command {
			case "fixstates":
				output, err = runZPushAdminCommand(ctx, "fixstates", "-u", username)
			case "clearloop":
				output, err = runZPushAdminCommand(ctx, "clearloop", "-u", username, "-d", deviceID)
			case "resync":
				output, err = runZPushAdminCommand(ctx, "resync", "-u", username, "-d", deviceID)
			case "remove":
				output, err = runZPushAdminCommand(ctx, "remove", "-u", username, "-d", deviceID)
			default:
				log.Printf("Unknown command: %s", command)
				continue
			}

			if err != nil {
				log.Printf("Error processing command '%s' for user '%s': %v", command, username, err)
				continue
			}

			log.Printf("Successfully processed command '%s' for user '%s'. Output: %s", command, username, output)
		}
	}
}

func isLocalUser(ctx context.Context, username string) (bool, error) {
	output, err := runZPushAdminCommand(ctx, "list", "-u", username)
	if err != nil {
		return false, err
	}
	// Z-Push error message when user is not found
	return !strings.Contains(output, "no devices found"), nil
}

func runZPushAdminCommand(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "z-push-admin", "-a", command)
	cmd.Args = append(cmd.Args, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
