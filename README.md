# vrpaws — Connect, share, and relive your VRChat moments with friends.

vrpaws is a small client/daemon for uploading VRChat photos to the vrpa.ws website.

[![Downloads](https://img.shields.io/github/downloads/vrpaws/daemon/total?color=6451f1)](https://github.com/vrpaws/daemon/releases/latest)

## Download and run

[![Download for Windows](https://img.shields.io/badge/Download-Windows-blue?logo=windows&style=for-the-badge)](https://github.com/vrpaws/daemon/releases/latest/download/vrpaws-client-latest.exe)

Windows (recommended)
1. Click the "Download for Windows" button above or visit the Releases page:
   https://github.com/vrpaws/daemon/releases/latest
2. Run the downloaded installer (vrpaws-client-*.exe) and follow the prompts.

> [!TIP]  
> You can add a shortcut to the executable file in VRCX to auto-launch the application everytime VRChat runs.

Linux / macOS (prebuilt binary)
1. Download the appropriate binary from the Releases page.
2. Make it executable and run:
```bash
curl -L -o vrpaws-client https://github.com/vrpaws/daemon/releases/latest/download/vrpaws-client-latest-linux-amd64
chmod +x vrpaws-client
./vrpaws-client
```

Sign up
- Create an account at https://vrpa.ws/ to enable uploads and manage your photos.

Troubleshooting & Support
- If uploads fail, check your network/firewall and that you are signed in at https://vrpa.ws/.
- For errors or to report bugs, open an issue: https://github.com/vrpaws/daemon/issues
- Include environment details (OS, client version) and any logs when filing an issue.
- Log files and settings are found in your AppData folder `%appdata%\VRPaws`

## Is vrpaws against VRChat's TOS?

**No.**

vrpaws is an external tool that uses the VRChat API to upload photos taken in VRChat to the vrpa.ws website. It does not modify the game or provide any in-game cheats or modifications. It looks at your pictures folder and uploads any new images that gets taken.

> [!NOTE]  
> vrpaws is not endorsed by VRChat and does not reflect the views of VRChat. VRChat and all associated properties are trademarks of VRChat © VRChat Inc.