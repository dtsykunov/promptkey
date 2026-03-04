package main

import (
	"fmt"
	"strings"
)

// parseHotkey parses a hotkey string like "ctrl+alt+`" into modifier flags and VK code.
// Modifier values match Windows MOD_* constants; VK values match Windows VK_* codes.
func parseHotkey(s string) (mods uint32, vk uint32, err error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch p {
		case "ctrl":
			mods |= 0x0002
		case "alt":
			mods |= 0x0001
		case "shift":
			mods |= 0x0004
		case "win":
			mods |= 0x0008
		default:
			switch p {
			case "`":
				vk = 0xC0
			case "-":
				vk = 0xBD
			case "=":
				vk = 0xBB
			case "[":
				vk = 0xDB
			case "]":
				vk = 0xDD
			case ";":
				vk = 0xBA
			case "'":
				vk = 0xDE
			case ",":
				vk = 0xBC
			case ".":
				vk = 0xBE
			case "/":
				vk = 0xBF
			case "\\":
				vk = 0xDC
			case "space":
				vk = 0x20
			case "f1":
				vk = 0x70
			case "f2":
				vk = 0x71
			case "f3":
				vk = 0x72
			case "f4":
				vk = 0x73
			case "f5":
				vk = 0x74
			case "f6":
				vk = 0x75
			case "f7":
				vk = 0x76
			case "f8":
				vk = 0x77
			case "f9":
				vk = 0x78
			case "f10":
				vk = 0x79
			case "f11":
				vk = 0x7A
			case "f12":
				vk = 0x7B
			default:
				if len(p) == 1 {
					c := p[0]
					if c >= 'a' && c <= 'z' {
						vk = uint32('A' + (c - 'a'))
					} else if c >= '0' && c <= '9' {
						vk = uint32(c)
					} else {
						return 0, 0, fmt.Errorf("unknown key token %q", p)
					}
				} else {
					return 0, 0, fmt.Errorf("unknown key token %q", p)
				}
			}
		}
	}
	if mods == 0 {
		return 0, 0, fmt.Errorf("hotkey must include at least one modifier (ctrl/alt/shift/win)")
	}
	if vk == 0 {
		return 0, 0, fmt.Errorf("hotkey must include a non-modifier key")
	}
	return mods, vk, nil
}
