package configuration

import (
	"strings"
)

const (
	DOCKER_PASSWORD_TAG_BEGIN = "<dockerPassword>"
	DOCKER_PASSWORD_TAG_END   = "</dockerPassword>"
	PASSWORD_TAG_BEGIN        = "<password>"
	PASSWORD_TAG_END          = "</password>"
	DOCKER_USERNAME_TAG_BEGIN = "<dockerUsername>"
	DOCKER_USERNAME_TAG_END   = "</dockerUsername>"
	USERNAME_TAG_BEGIN        = "<username>"
	USERNAME_TAG_END          = "</username>"
	MASK                      = "XXXXX"
)

func MaskSecurityInfo(payload string) string {
	masked := maskPassword(payload)
	return maskUsername(masked)
}

func maskPassword(payload string) string {
	payload = maskTag(payload, DOCKER_PASSWORD_TAG_BEGIN, DOCKER_PASSWORD_TAG_END)
	payload = maskTag(payload, PASSWORD_TAG_BEGIN, PASSWORD_TAG_END)
	return payload
}

func maskUsername(payload string) string {
	payload = maskTag(payload, DOCKER_USERNAME_TAG_BEGIN, DOCKER_USERNAME_TAG_END)
	payload = maskTag(payload, USERNAME_TAG_BEGIN, USERNAME_TAG_END)
	return payload
}

func maskTag(payload, tagBegin, tagEnd string) string {
	for {
		start := strings.Index(payload, tagBegin)
		if start == -1 {
			break
		}
		end := strings.Index(payload, tagEnd)
		if end == -1 {
			break
		}
		end += len(tagEnd)
		payload = payload[:start+len(tagBegin)] + MASK + payload[end-len(tagEnd):]
	}
	return payload
}
