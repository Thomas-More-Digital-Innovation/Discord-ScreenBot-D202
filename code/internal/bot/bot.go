package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Thomas-More-Digital-Innovation/Discord-ScreenBot-D202/internal/amx"
	"github.com/Thomas-More-Digital-Innovation/Discord-ScreenBot-D202/internal/config"
	"github.com/bwmarrin/discordgo"
)

// Bot manages the Discord bot session and slash command interactions.
type Bot struct {
	cfg        *config.Config
	amxClient  amx.Client
	session    *discordgo.Session
	appID      string
	commands   []*discordgo.ApplicationCommand
	registered []*discordgo.ApplicationCommand
}

// New creates a new Discord Bot instance.
func New(cfg *config.Config, amxClient amx.Client) (*Bot, error) {
	if cfg.Discord.Token == "" {
		return nil, fmt.Errorf("discord bot token is required")
	}

	token := strings.TrimSpace(cfg.Discord.Token)
	if token == "" {
		return nil, fmt.Errorf("discord bot token is empty")
	}
	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}

	session, err := discordgo.New(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	b := &Bot{
		cfg:       cfg,
		amxClient: amxClient,
		session:   session,
		appID:     cfg.Discord.AppID,
	}

	b.setupCommands()
	return b, nil
}

func (b *Bot) setupCommands() {
	b.commands = []*discordgo.ApplicationCommand{
		{
			Name:        "video",
			Description: "Route an input source to a screen output on the AMX matrix switcher",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "screen",
					Description:  "Target screen / output port",
					Required:     true,
					Autocomplete: true,
				},
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "input",
					Description:  "Video source / input port",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "screens",
			Description: "List all configured screen outputs and video inputs",
		},
		{
			Name:        "ping",
			Description: "Check bot and AMX switcher connectivity",
		},
	}
}

// Start opens the Discord websocket connection and registers slash commands.
func (b *Bot) Start() error {
	b.session.AddHandler(b.handleInteraction)
	b.session.AddHandler(b.handleMessage)

	b.session.Identify.Intents = discordgo.IntentsGuildMessages

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open discord websocket: %w", err)
	}

	// Auto-detect AppID if not provided
	if b.appID == "" {
		b.appID = b.session.State.User.ID
	}

	log.Printf("Logged in as Discord user: %s#%s (ID: %s)",
		b.session.State.User.Username, b.session.State.User.Discriminator, b.appID)

	if b.cfg.Discord.ChannelID != "" {
		log.Printf("Channel restriction active: only responding in channel ID %s", b.cfg.Discord.ChannelID)
	}

	guildID := b.cfg.Discord.GuildID
	if guildID != "" {
		// Clean up any stale global commands so they don't shadow guild commands
		log.Printf("Clearing stale global slash commands...")
		_, _ = b.session.ApplicationCommandBulkOverwrite(b.appID, "", []*discordgo.ApplicationCommand{})

		log.Printf("Registering slash commands for Guild ID: %s (instant availability)", guildID)
		createdCmds, err := b.session.ApplicationCommandBulkOverwrite(b.appID, guildID, b.commands)
		if err != nil {
			return fmt.Errorf("failed to register guild slash commands: %w", err)
		}
		b.registered = createdCmds
		for _, cmd := range createdCmds {
			log.Printf("Registered slash command: /%s", cmd.Name)
		}
	} else {
		log.Printf("Registering slash commands globally (propagation may take up to 1 hour)...")
		createdCmds, err := b.session.ApplicationCommandBulkOverwrite(b.appID, "", b.commands)
		if err != nil {
			return fmt.Errorf("failed to register global slash commands: %w", err)
		}
		b.registered = createdCmds
		for _, cmd := range createdCmds {
			log.Printf("Registered slash command: /%s", cmd.Name)
		}
	}

	return nil
}

// Stop gracefully shuts down the Discord bot and cleans up handlers.
func (b *Bot) Stop() error {
	log.Println("Shutting down Discord bot...")
	return b.session.Close()
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Restrict to designated channel if channel_id is specified; ignore others completely
	if b.cfg.Discord.ChannelID != "" && i.ChannelID != b.cfg.Discord.ChannelID {
		return
	}

	// Handle live autocomplete suggestions
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		b.handleAutocomplete(s, i)
		return
	}

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	user := "unknown"
	if i.Member != nil && i.Member.User != nil {
		user = i.Member.User.Username
	} else if i.User != nil {
		user = i.User.Username
	}
	log.Printf("Received slash command /%s in channel %s from %s", data.Name, i.ChannelID, user)

	// Defer response to prevent Discord 3-second timeout during network calls
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error deferring interaction: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var responseEmbed *discordgo.MessageEmbed

	switch data.Name {
	case "video":
		responseEmbed = b.handleVideoCommand(ctx, data)
	case "screens":
		responseEmbed = b.handleScreensCommand()
	case "ping":
		responseEmbed = b.handlePingCommand(ctx)
	default:
		responseEmbed = &discordgo.MessageEmbed{
			Title:       "Unknown Command",
			Description: fmt.Sprintf("Command `/%s` is not recognized.", data.Name),
			Color:       0xe74c3c, // Red
		}
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{responseEmbed},
	})
	if err != nil {
		log.Printf("Error editing interaction response: %v", err)
	}
}

// handleMessage supports regular text messages like "/video screena input1" or "!video screena input1"
func (b *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}

	content := strings.TrimSpace(m.Content)
	if content == "" {
		return
	}

	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	// Support mentions: @ScreenBot /video screena input1
	if strings.HasPrefix(cmd, "<@") && len(parts) > 1 {
		parts = parts[1:]
		cmd = strings.ToLower(parts[0])
	}

	trimmedCmd := strings.TrimPrefix(strings.TrimPrefix(cmd, "/"), "!")

	// Only process recognized bot commands
	if trimmedCmd != "video" && trimmedCmd != "screens" && trimmedCmd != "ping" {
		return
	}

	log.Printf("Received chat message command '%s' in channel %s from %s", content, m.ChannelID, m.Author.Username)

	// Check channel restriction
	if b.cfg.Discord.ChannelID != "" && m.ChannelID != b.cfg.Discord.ChannelID {
		log.Printf("Ignored message '%s': sent in channel %s, but restricted to channel %s",
			content, m.ChannelID, b.cfg.Discord.ChannelID)
		return
	}

	switch trimmedCmd {
	case "video":
		if len(parts) < 3 {
			s.ChannelMessageSend(m.ChannelID, "⚠️ Usage: `/video <screen> <input>` (e.g. `/video screena input1`)")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		embed := b.executeSwitch(ctx, parts[1], parts[2])
		s.ChannelMessageSendEmbed(m.ChannelID, embed)

	case "screens":
		embed := b.handleScreensCommand()
		s.ChannelMessageSendEmbed(m.ChannelID, embed)

	case "ping":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		embed := b.handlePingCommand(ctx)
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
	}
}

func (b *Bot) handleVideoCommand(ctx context.Context, data discordgo.ApplicationCommandInteractionData) *discordgo.MessageEmbed {
	var rawScreen, rawInput string

	for _, opt := range data.Options {
		switch opt.Name {
		case "screen":
			rawScreen = opt.StringValue()
		case "input":
			rawInput = opt.StringValue()
		}
	}

	return b.executeSwitch(ctx, rawScreen, rawInput)
}

func (b *Bot) executeSwitch(ctx context.Context, rawScreen, rawInput string) *discordgo.MessageEmbed {
	outputID, outName, err := b.cfg.ResolveOutput(rawScreen)
	if err != nil {
		return &discordgo.MessageEmbed{
			Title:       "❌ Invalid Screen / Output",
			Description: err.Error(),
			Color:       0xe74c3c,
		}
	}

	inputID, inName, err := b.cfg.ResolveInput(rawInput)
	if err != nil {
		return &discordgo.MessageEmbed{
			Title:       "❌ Invalid Video Input",
			Description: err.Error(),
			Color:       0xe74c3c,
		}
	}

	// Send switch command to AMX matrixer
	err = b.amxClient.SwitchVideo(ctx, outputID, inputID)
	if err != nil {
		log.Printf("AMX Switch error: %v", err)
		return &discordgo.MessageEmbed{
			Title:       "⚠️ AMX Switcher Error",
			Description: fmt.Sprintf("Failed to route video on AMX matrixer:\n```\n%s\n```", err.Error()),
			Color:       0xe74c3c,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Target: %s", b.amxClient.GetHost()),
			},
		}
	}

	return &discordgo.MessageEmbed{
		Title:       "🎬 Video Routed Successfully",
		Description: fmt.Sprintf("Successfully connected **%s** to **%s**.", inName, outName),
		Color:       0x2ecc71, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Target Screen (Output)",
				Value:  fmt.Sprintf("**%s** (Port %d)", outName, outputID),
				Inline: true,
			},
			{
				Name:   "Video Source (Input)",
				Value:  fmt.Sprintf("**%s** (Port %d)", inName, inputID),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AMX Enova DVX-3150HD-SP",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (b *Bot) handleScreensCommand() *discordgo.MessageEmbed {
	outNames := b.cfg.GetOutputNames()
	inNames := b.cfg.GetInputNames()

	var outLines []string
	for _, name := range outNames {
		outLines = append(outLines, fmt.Sprintf("• `%s` → Output Port %d", name, b.cfg.Mapping.Outputs[name]))
	}

	var inLines []string
	for _, name := range inNames {
		inLines = append(inLines, fmt.Sprintf("• `%s` → Input Port %d", name, b.cfg.Mapping.Inputs[name]))
	}

	return &discordgo.MessageEmbed{
		Title:       "📺 Configured Screens & Inputs",
		Description: "Aliases currently mapped in configuration:",
		Color:       0x3498db, // Blue
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Screen Outputs",
				Value:  strings.Join(outLines, "\n"),
				Inline: false,
			},
			{
				Name:   "Video Inputs",
				Value:  strings.Join(inLines, "\n"),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("AMX Host: %s", b.amxClient.GetHost()),
		},
	}
}

func (b *Bot) handlePingCommand(ctx context.Context) *discordgo.MessageEmbed {
	start := time.Now()
	err := b.amxClient.Ping(ctx)
	latency := time.Since(start).Round(time.Millisecond)

	amxStatus := fmt.Sprintf("✅ Online (%s)", latency)
	color := 0x2ecc71 // Green

	if err != nil {
		amxStatus = fmt.Sprintf("❌ Offline / Unreachable (%v)", err)
		color = 0xe74c3c // Red
	}

	return &discordgo.MessageEmbed{
		Title: "🏓 Pong!",
		Color: color,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Discord Gateway",
				Value:  "✅ Connected",
				Inline: true,
			},
			{
				Name:   "AMX DVX Switcher",
				Value:  amxStatus,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("AMX Host: %s", b.amxClient.GetHost()),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (b *Bot) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0)

	for _, opt := range data.Options {
		if !opt.Focused {
			continue
		}

		query := strings.ToLower(strings.TrimSpace(opt.StringValue()))

		switch opt.Name {
		case "screen":
			for _, c := range b.cfg.GetOutputChoices() {
				if query == "" || strings.Contains(strings.ToLower(c.Name), query) || strings.Contains(strings.ToLower(c.Value), query) {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  c.Name,
						Value: c.Value,
					})
					if len(choices) >= 25 {
						break
					}
				}
			}
		case "input":
			for _, c := range b.cfg.GetInputChoices() {
				if query == "" || strings.Contains(strings.ToLower(c.Name), query) || strings.Contains(strings.ToLower(c.Value), query) {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  c.Name,
						Value: c.Value,
					})
					if len(choices) >= 25 {
						break
					}
				}
			}
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}
