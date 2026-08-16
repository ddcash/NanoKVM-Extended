#ifndef OLED_CONFIG_H_
#define OLED_CONFIG_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
 * Which rows the OLED shows, read from /etc/kvm/oled.conf.
 *
 * The file lives outside /kvmapp so the choice survives an application update,
 * the same reason the HID mode marker does. It is plain key=value rather than
 * JSON so this daemon needs no parser:
 *
 *   show_ip=1
 *   show_res=1
 *   show_type=1
 *   show_stream=1
 *   show_quality=1
 *   title=Rack KVM
 *
 * A missing file means "show everything", which is the stock behaviour.
 */
#define OLED_CONFIG_FILE   "/etc/kvm/oled.conf"
#define OLED_TITLE_MAX     20

typedef struct {
    uint8_t show_ip;
    uint8_t show_res;
    uint8_t show_type;
    uint8_t show_stream;
    uint8_t show_quality;
    char    title[OLED_TITLE_MAX + 1];
} oled_config_t;

/*
 * Current configuration. Re-reads the file when its timestamp changes, so an
 * edit takes effect without restarting the daemon. Never returns NULL.
 */
const oled_config_t* oled_config_get(void);

/* Whether the row belonging to a KVM_* display state should be drawn. */
uint8_t oled_config_row_enabled(uint8_t kvm_state);

#ifdef __cplusplus
}
#endif

#endif // OLED_CONFIG_H_
