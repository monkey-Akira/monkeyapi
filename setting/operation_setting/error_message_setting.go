package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type ErrorMessageSetting struct {
	Enabled  bool              `json:"enabled"`
	Mappings map[string]string `json:"mappings"`
}

var errorMessageSetting = ErrorMessageSetting{
	Enabled: false,
	Mappings: map[string]string{
		"insufficient_user_quota":        "\u989d\u5ea6\u4e0d\u8db3\uff0c\u8bf7\u5145\u503c\u540e\u518d\u8bd5\u3002",
		"pre_consume_token_quota_failed": "\u989d\u5ea6\u9884\u6263\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002",
		"channel:no_available_key":       "\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\uff0c\u8bf7\u7a0d\u540e\u518d\u8bd5\u3002",
		"channel:invalid_key":            "\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\uff0c\u8bf7\u8054\u7cfb\u7ba1\u7406\u5458\u5904\u7406\u3002",
		"channel:model_mapped_error":     "\u6a21\u578b\u914d\u7f6e\u5f02\u5e38\uff0c\u8bf7\u8054\u7cfb\u7ba1\u7406\u5458\u5904\u7406\u3002",
		"do_request_failed":              "\u8fde\u63a5\u4e0a\u6e38\u670d\u52a1\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002",
		"bad_response_status_code":       "\u4e0a\u6e38\u670d\u52a1\u54cd\u5e94\u5f02\u5e38\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002",
		"bad_response_body":              "\u4e0a\u6e38\u670d\u52a1\u8fd4\u56de\u5f02\u5e38\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002",
		"model_not_found":                "\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\u3002",
		"prompt_blocked":                 "\u8bf7\u6c42\u5185\u5bb9\u672a\u901a\u8fc7\u5b89\u5168\u68c0\u67e5\uff0c\u8bf7\u8c03\u6574\u540e\u91cd\u8bd5\u3002",
		"sensitive_words_detected":       "\u8bf7\u6c42\u5185\u5bb9\u5305\u542b\u654f\u611f\u8bcd\uff0c\u8bf7\u8c03\u6574\u540e\u91cd\u8bd5\u3002",
		"invalid_request":                "\u8bf7\u6c42\u53c2\u6570\u6709\u8bef\uff0c\u8bf7\u68c0\u67e5\u540e\u91cd\u8bd5\u3002",
	},
}

func init() {
	config.GlobalConfig.Register("error_message_setting", &errorMessageSetting)
}

func GetErrorMessageSetting() *ErrorMessageSetting {
	return &errorMessageSetting
}
