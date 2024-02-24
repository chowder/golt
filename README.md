# ⚔️ golt

**golt** is a CLI-based third-party game launcher for OldSchool Runescape.

# Why?

[The official recommendation](https://help.jagex.com/hc/en-gb/articles/13413514881937-Downloading-the-Jagex-Launcher-on-Linux)
points to several community projects, most of which involve running the launcher in Wine or a similar environment. However, I really wanted a native solution, or one
that didn't require installing a 1.5GB compatibility layer solely for launcher functionality.

While there is a project that served as the inspiration for this one, and functions seamlessly on Linux, there were some boxes that it didn't tick for me:

- Its installation size is ~460MB
- Lack of support for overriding the client launch command/environment
- It's a bit of a pain to compile

In comparison, the linux-amd64 build for `golt` is a single 6.8MB binary.

# Installation

Either download the latest binary from the [Releases](https://github.com/chowder/golt/releases) page (or build it yourself), and add it to a directory on your `PATH`.

Then, create a desktop entry for the application:

```
[Desktop Entry]
Type=Application
Name=golt
Exec=golt %U
StartupNotify=true
Terminal=true
MimeType=x-scheme-handler/jagex;
```

Then, register it to be the default handler for the `jagex:<...>` scheme:

```
xdg-mime default golt.desktop x-scheme-handler/jagex
```

Finally, set up an iptable entry to redirect `localhost:80` to `localhost:8080`:

```
sudo iptables -t nat -I OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-ports 8080
```

<details>
    <summary>Why?</summary>

The login flow is currently done in the browser:

- The OAuth login redirects to a page which invokes a scheme handler
- The game login step redirects to `http://localhost`

These redirect URLs are validated server side, so cannot be modified on the client side. 

As for the iptable entry, most Linux distros don't allow binding to port 80, so `golt` binds to port 8080 instead. 

</details>

# Configuration

| Environment Variable | Default             | Description                                                                                                                                                    |
|----------------------|---------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `GOLT_GAME_PATH`     | `RuneLite.AppImage` | Either a binary on `PATH`, or an absolute path to the client to launch.<br/>This value is passed to `exec.Command`([docs](https://pkg.go.dev/os/exec#Command)) |

# Disclaimer

This project is not affiliated with or endorsed by Jagex Ltd. It is an independent project created for educational and
testing purposes. Please use responsibly and adhere to Jagex's terms of service.