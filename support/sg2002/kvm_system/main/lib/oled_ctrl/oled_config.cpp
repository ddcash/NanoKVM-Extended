#include "oled_config.h"
#include "oled_ctrl.h"

#include <stdio.h>
#include <string.h>
#include <sys/stat.h>

static oled_config_t config = {1, 1, 1, 1, 1, ""};
static time_t config_mtime = 0;
static uint8_t config_loaded = 0;

static void trim(char* text)
{
    size_t len = strlen(text);
    while (len > 0 && (text[len - 1] == '\n' || text[len - 1] == '\r' ||
                       text[len - 1] == ' '  || text[len - 1] == '\t')) {
        text[--len] = '\0';
    }
}

/* Named distinctly: a plain apply() resolves to std::apply, which the Maix
   headers pull into scope, and the call then fails inside <tuple>. */
static void oled_apply_setting(const char* key, const char* value)
{
    /* Anything other than an explicit 0 counts as on, so a malformed value
       shows the row rather than silently hiding it. */
    uint8_t on = (uint8_t)(strcmp(value, "0") != 0);

    if (strcmp(key, "show_ip") == 0)           config.show_ip = on;
    else if (strcmp(key, "show_res") == 0)     config.show_res = on;
    else if (strcmp(key, "show_type") == 0)    config.show_type = on;
    else if (strcmp(key, "show_stream") == 0)  config.show_stream = on;
    else if (strcmp(key, "show_quality") == 0) config.show_quality = on;
    else if (strcmp(key, "title") == 0) {
        strncpy(config.title, value, OLED_TITLE_MAX);
        config.title[OLED_TITLE_MAX] = '\0';
    }
}

static void reload(void)
{
    /* Defaults first, so a key dropped from the file reverts to shown rather
       than keeping whatever the previous read left behind. */
    config.show_ip = 1;
    config.show_res = 1;
    config.show_type = 1;
    config.show_stream = 1;
    config.show_quality = 1;
    config.title[0] = '\0';

    FILE* file = fopen(OLED_CONFIG_FILE, "r");
    if (file == NULL) {
        return;
    }

    char line[128];
    while (fgets(line, sizeof(line), file) != NULL) {
        trim(line);
        if (line[0] == '\0' || line[0] == '#') {
            continue;
        }

        char* separator = strchr(line, '=');
        if (separator == NULL) {
            continue;
        }
        *separator = '\0';

        oled_apply_setting(line, separator + 1);
    }

    fclose(file);
}

const oled_config_t* oled_config_get(void)
{
    struct stat info;
    if (stat(OLED_CONFIG_FILE, &info) != 0) {
        /* No file: stock behaviour. Reload once so a deleted file restores the
           defaults rather than keeping the last values read. */
        if (config_loaded && config_mtime != 0) {
            reload();
            config_mtime = 0;
        }
        config_loaded = 1;
        return &config;
    }

    if (!config_loaded || info.st_mtime != config_mtime) {
        reload();
        config_mtime = info.st_mtime;
        config_loaded = 1;
    }

    return &config;
}

uint8_t oled_config_row_enabled(uint8_t kvm_state)
{
    const oled_config_t* current = oled_config_get();

    switch (kvm_state) {
        case KVM_ETH_IP:
        case KVM_WIFI_IP:
            return current->show_ip;
        case KVM_HDMI_RES:
            return current->show_res;
        case KVM_STEAM_TYPE:
            return current->show_type;
        case KVM_STEAM_FPS:
            return current->show_stream;
        case KVM_JPG_QLTY:
            return current->show_quality;
        default:
            /* KVM_INIT and anything else always draws: the init pass paints the
               labels, and hiding an unknown state would blank the screen. */
            return 1;
    }
}
