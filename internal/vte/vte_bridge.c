#include "vte.h"
#include <pango/pango.h>

// Spawn callback: bridges VTE async result back to Go.
static void vteSpawnBridge(VteTerminal *terminal, GPid pid, GError *error, gpointer user_data) {
    int callbackID = (int)(intptr_t)user_data;
    int ptyFd = -1;
    char *errMsg = NULL;

    if (error != NULL) {
        errMsg = error->message;
    } else {
        VtePty *pty = vte_terminal_get_pty(terminal);
        if (pty != NULL) {
            ptyFd = vte_pty_get_fd(pty);
        }
    }

    goVteSpawnDone(callbackID, (int)pid, ptyFd, errMsg);
}

// Spawn a shell in the terminal asynchronously.
// callbackID is passed back to goVteSpawnDone.
void vteSpawnAsync(VteTerminal *terminal, const char *workingDir,
                   char **argv, int callbackID) {
    vte_terminal_spawn_async(
        terminal,
        VTE_PTY_DEFAULT,
        workingDir,
        argv,
        NULL,              // envv — inherit
        G_SPAWN_SEARCH_PATH,
        NULL, NULL, NULL,  // child_setup / data / destroy
        -1,                // timeout: -1 = default
        NULL,              // cancellable
        vteSpawnBridge,
        (gpointer)(intptr_t)callbackID
    );
}

// child-exited signal callback.
static void vteChildExitedBridge(VteTerminal *terminal, int status, gpointer user_data) {
    (void)terminal;
    goVteChildExited((int)(intptr_t)user_data, status);
}

// Connect child-exited signal for the given tabID.
void vteConnectChildExited(VteTerminal *terminal, int tabID) {
    g_signal_connect(G_OBJECT(terminal), "child-exited",
                     G_CALLBACK(vteChildExitedBridge),
                     (gpointer)(intptr_t)tabID);
}

// Write bytes to the terminal PTY.
void vteFeedChild(VteTerminal *terminal, const char *data, int len) {
    vte_terminal_feed_child(terminal, data, (gssize)len);
}

// Set the terminal font from a Pango font description string (e.g. "Monospace 12").
void vteSetFont(VteTerminal *terminal, const char *desc_str) {
    PangoFontDescription *desc = pango_font_description_from_string(desc_str);
    if (desc == NULL) {
        return;
    }
    vte_terminal_set_font(terminal, desc);
    pango_font_description_free(desc);
}
