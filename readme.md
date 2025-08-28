# MessageMerger - Mattermost Plugin

A Mattermost plugin that automatically aggregates confirmation messages, merging multiple user acknowledgments into a single message to reduce channel noise.

## Features

- 🎯 **Auto Aggregation**: Automatically merges identical confirmation messages from multiple users
- 🔧 **Configurable Trigger Words**: Support custom trigger words (default: 收到, 已收到, 确认)
- 📊 **User List Display**: Clear display of all confirmed users
- 🚫 **Duplicate Prevention**: Prevents duplicate entries for the same user
- ⚡ **Real-time Updates**: Messages update in real-time without page refresh

## How It Works

1. User A sends "收到" → Message displays: "收到"
2. User B sends "收到" → Message updates to: "收到 -- UserB"
3. User C sends "收到" → Message updates to: "收到 -- UserB, UserC"
4. User B sends "收到" again → Message unchanged (duplicate prevention)

## Installation

### Prerequisites

- Go 1.16 or later
- Mattermost Server v5.20.0 or later

### Build the Plugin

```bash
# Clone the repository
git clone <repository-url>
cd MsgMearge

# Initialize build tools
make apply

# Build the plugin
make dist
```

### Install to Mattermost

1. Log in to Mattermost as a System Administrator
2. Go to **System Console** → **Plugins** → **Management**
3. Click **Upload Plugin** and select the generated `dist/MessageMerger-1.0.0.tar.gz`
4. Enable the plugin

## Configuration

In the plugin management page, find **Message Aggregator** and click **Settings**:

- **Trigger Words**: Set trigger keywords separated by commas
  - Default: `收到,已收到,确认`
  - Example: `收到,已收到,确认,OK,好的`

- **Max Lookback Messages**: Set the range for searching identical messages
  - Default: `5`
  - Recommended: 5-20

- **Reject Message**: Set the message shown when a duplicate is prevented
  - Default: `消息已合并,无需重复发送`

## Usage Examples

### Basic Usage

```
Admin: Please confirm receipt of this notice
UserA: 收到
UserB: 收到
UserC: 收到
```

Result:
```
Admin: Please confirm receipt of this notice
System: 收到 -- UserA, UserB, UserC
```

### Custom Trigger Words

If configured with trigger words `收到,确认,OK`:

```
Admin: Meeting time confirmation
UserA: 确认
UserB: OK
UserC: 收到
```

Result:
```
Admin: Meeting time confirmation
System: 确认 -- UserA
System: OK -- UserB
System: 收到 -- UserC
```

## Development

### Local Development

```bash
# Install dependencies
go mod tidy

# Run tests
make test

# Build development version
make debug-dist

# Deploy to development server
make deploy
```

### Build Commands

```bash
# Build for all platforms
make dist

# Build server only
make server

# Clean build artifacts
make clean

# Check code style
make check-style
```

## Technical Details

### Core Logic

1. **Message Monitoring**: Uses `MessageWillBePosted` hook to listen for new messages
2. **Trigger Word Matching**: Checks if message exactly matches configured trigger words
3. **Historical Message Search**: Searches recent messages for identical content
4. **Message Aggregation**: Updates existing message or creates new aggregated message
5. **Duplicate Detection**: Prevents same user from appearing multiple times

### File Structure

```
MsgMearge/
├── plugin.json          # Plugin configuration and metadata
├── plugin.go           # Main plugin logic
├── go.mod              # Go module dependencies
├── go.sum              # Go module checksums
├── makefile            # Build scripts
├── build.ps1           # PowerShell build script
├── dist/               # Build output directory
├── readme.md           # Chinese documentation
├── windows_setup_guide.md  # Windows setup guide
└── README.md           # English documentation
```

### API Hooks Used

- `MessageWillBePosted`: Monitors new messages before posting
- `OnConfigurationChange`: Handles configuration updates
- `GetPostsForChannel`: Retrieves channel history
- `UpdatePost`: Updates message content
- `SendEphemeralPost`: Sends temporary system messages

## Troubleshooting

### Q: Plugin not working?
A:
1. Check if plugin is enabled
2. Verify trigger word configuration
3. Check Mattermost server logs for errors

### Q: What message types are supported?
A: Currently only supports plain text messages, not messages with attachments, emojis, or formatting.

### Q: Can it support multiple languages?
A: Yes, by modifying the trigger words configuration to support different language confirmation words.

### Q: Are there limits on message aggregation?
A: No hard limits, but keep reasonable numbers for display quality.

## License

This project is licensed under the MIT License.

## Contributing

Issues and Pull Requests are welcome to improve this plugin!

## Changelog

### v1.0.0
- Initial release
- Basic message aggregation functionality
- Configurable trigger words and lookback range
- Duplicate user prevention