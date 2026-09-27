//go:build windows

package wintray

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_ = iota
	openAppsMenuID
	settingsUIMenuID
	loginMenuID
	loggedInMenuID
	logoutMenuID
	accountSeparatorMenuID
	updateSeparatorMenuID
	updateAvailableMenuID
	updateMenuID
	separatorMenuID
	diagLogsMenuID
	diagSeparatorMenuID
	quitMenuID
)

func (t *winTray) initMenus() error {
	if err := t.addOrUpdateMenuItem(openAppsMenuID, 0, openAppsMenuTitle, false); err != nil {
		return fmt.Errorf("unable to create menu entries %w", err)
	}
	if err := t.addOrUpdateMenuItem(settingsUIMenuID, 0, settingsUIMenuTitle, false); err != nil {
		return fmt.Errorf("unable to create menu entries %w", err)
	}
	if err := t.addOrUpdateMenuItem(loginMenuID, 0, loginMenuTitle, false); err != nil {
		return fmt.Errorf("unable to create menu entries %w", err)
	}
	if err := t.addSeparatorMenuItem(accountSeparatorMenuID, 0); err != nil {
		return fmt.Errorf("unable to create menu entries %w", err)
	}
	if err := t.addOrUpdateMenuItem(diagLogsMenuID, 0, diagLogsMenuTitle, false); err != nil {
		return fmt.Errorf("unable to create menu entries %w\n", err)
	}
	if err := t.addSeparatorMenuItem(diagSeparatorMenuID, 0); err != nil {
		return fmt.Errorf("unable to create menu entries %w", err)
	}

	if err := t.addOrUpdateMenuItem(quitMenuID, 0, quitMenuTitle, false); err != nil {
		return fmt.Errorf("unable to create menu entries %w", err)
	}
	return nil
}

func (t *winTray) removeMenuItem(menuItemID uint32) error {
	t.muMenus.RLock()
	menu := t.menus[0]
	t.muMenus.RUnlock()

	ret, _, err := pDeleteMenu.Call(uintptr(menu), uintptr(MF_BYCOMMAND), uintptr(menuItemID))
	if ret == 0 {
		return fmt.Errorf("failed to delete menu item %d: %w", menuItemID, err)
	}

	t.delFromVisibleItems(0, menuItemID)
	return nil
}

// refreshAuthMenu rebuilds the account section of the tray menu.
func (t *winTray) refreshAuthMenu() {
	name, signedIn := t.app.AuthState()

	if signedIn {
		if err := t.removeMenuItem(loginMenuID); err != nil {
			slog.Debug("failed to remove login menu item", "error", err)
		}
		if err := t.addOrUpdateMenuItem(loggedInMenuID, 0, fmt.Sprintf(loggedInMenuTitle, name), true); err != nil {
			slog.Error("failed to update account menu", "error", err)
		}
		if err := t.addOrUpdateMenuItem(logoutMenuID, 0, logoutMenuTitle, false); err != nil {
			slog.Error("failed to update account menu", "error", err)
		}
		return
	}

	if err := t.removeMenuItem(loggedInMenuID); err != nil {
		slog.Debug("failed to remove signed-in menu item", "error", err)
	}
	if err := t.removeMenuItem(logoutMenuID); err != nil {
		slog.Debug("failed to remove logout menu item", "error", err)
	}
	if err := t.addOrUpdateMenuItem(loginMenuID, 0, loginMenuTitle, false); err != nil {
		slog.Error("failed to update account menu", "error", err)
	}
}

func (t *winTray) UpdateAvailable(ver string) error {
	if !t.updateNotified {
		slog.Debug("updating menu and sending notification for new update")
		if err := t.addSeparatorMenuItem(updateSeparatorMenuID, 0); err != nil {
			return fmt.Errorf("unable to create menu entries %w", err)
		}
		if err := t.addOrUpdateMenuItem(updateAvailableMenuID, 0, updateAvailableMenuTitle, true); err != nil {
			return fmt.Errorf("unable to create menu entries %w", err)
		}
		if err := t.addOrUpdateMenuItem(updateMenuID, 0, updateMenuTitle, false); err != nil {
			return fmt.Errorf("unable to create menu entries %w", err)
		}
		if err := t.addSeparatorMenuItem(separatorMenuID, 0); err != nil {
			return fmt.Errorf("unable to create menu entries %w", err)
		}
		iconFilePath, err := iconBytesToFilePath(wt.updateIcon)
		if err != nil {
			return fmt.Errorf("unable to write icon data to temp file: %w", err)
		}
		if err := wt.setIcon(iconFilePath); err != nil {
			return fmt.Errorf("unable to set icon: %w", err)
		}
		t.updateNotified = true

		t.pendingUpdate = true
		// Now pop up the notification
		t.muNID.Lock()
		defer t.muNID.Unlock()
		copy(t.nid.InfoTitle[:], windows.StringToUTF16(updateTitle))
		copy(t.nid.Info[:], windows.StringToUTF16(fmt.Sprintf(updateMessage, ver)))
		t.nid.Flags |= NIF_INFO
		t.nid.Timeout = 10
		t.nid.Size = uint32(unsafe.Sizeof(*wt.nid))
		err = t.nid.modify()
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *winTray) showLogs() error {
	localAppData := os.Getenv("LOCALAPPDATA")
	AppDataDir := filepath.Join(localAppData, "Susan")
	cmd_path := "c:\\Windows\\system32\\cmd.exe"
	slog.Debug(fmt.Sprintf("viewing logs with start %s", AppDataDir))
	cmd := exec.Command(cmd_path, "/c", "start", AppDataDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false, CreationFlags: 0x08000000}
	err := cmd.Start()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to open log dir: %s", err))
	}
	return nil
}
