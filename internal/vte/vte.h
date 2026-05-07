#pragma once
#include <vte/vte.h>

// Exported Go functions called from C callbacks.
extern void goVteSpawnDone(int callbackID, int pid, int ptyFd, char *errMsg);
extern void goVteChildExited(int tabID, int status);

// Bridge functions defined in vte_bridge.c.
void vteSpawnAsync(VteTerminal *terminal, const char *workingDir, char **argv, int callbackID);
void vteConnectChildExited(VteTerminal *terminal, int tabID);
void vteFeedChild(VteTerminal *terminal, const char *data, int len);
void vteSetFont(VteTerminal *terminal, const char *desc_str);
