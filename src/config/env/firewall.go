package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

var (
	RateLimitEnabled bool

	RateLimitRegistration       int
	RateLimitRegistrationWindow time.Duration

	RateLimitLogin       int
	RateLimitLoginWindow time.Duration

	RateLimitPasswordReset       int
	RateLimitPasswordResetWindow time.Duration

	RateLimitEmailVerification       int
	RateLimitEmailVerificationWindow time.Duration

	RateLimitResetPassword       int
	RateLimitResetPasswordWindow time.Duration

	IPAllowlist []string
	IPDenylist  []string
)

func loadFirewallEnv() {
	RateLimitEnabled = os.Getenv("RATE_LIMIT_ENABLED") != "false"

	RateLimitRegistration = parseFirewallMax(os.Getenv("RATE_LIMIT_REGISTRATION"), 5)
	RateLimitRegistrationWindow = parseFirewallDuration(os.Getenv("RATE_LIMIT_REGISTRATION_WINDOW"), 1*time.Hour)

	RateLimitLogin = parseFirewallMax(os.Getenv("RATE_LIMIT_LOGIN"), 10)
	RateLimitLoginWindow = parseFirewallDuration(os.Getenv("RATE_LIMIT_LOGIN_WINDOW"), 15*time.Minute)

	RateLimitPasswordReset = parseFirewallMax(os.Getenv("RATE_LIMIT_PASSWORD_RESET"), 5)
	RateLimitPasswordResetWindow = parseFirewallDuration(os.Getenv("RATE_LIMIT_PASSWORD_RESET_WINDOW"), 1*time.Hour)

	RateLimitEmailVerification = parseFirewallMax(os.Getenv("RATE_LIMIT_EMAIL_VERIFICATION"), 5)
	RateLimitEmailVerificationWindow = parseFirewallDuration(os.Getenv("RATE_LIMIT_EMAIL_VERIFICATION_WINDOW"), 1*time.Hour)

	RateLimitResetPassword = parseFirewallMax(os.Getenv("RATE_LIMIT_RESET_PASSWORD"), 10)
	RateLimitResetPasswordWindow = parseFirewallDuration(os.Getenv("RATE_LIMIT_RESET_PASSWORD_WINDOW"), 1*time.Hour)

	IPAllowlist = parseFirewallCIDRList(os.Getenv("IP_ALLOWLIST"))
	IPDenylist = parseFirewallCIDRList(os.Getenv("IP_DENYLIST"))

	if RateLimitEnabled {
		pterm.DefaultLogger.Info("Rate limiting is ENABLED")
	} else {
		pterm.DefaultLogger.Info("Rate limiting is DISABLED")
	}

	if len(IPAllowlist) > 0 {
		pterm.DefaultLogger.Info(fmt.Sprintf("IP allowlist active: %d CIDR(s)", len(IPAllowlist)))
	}
	if len(IPDenylist) > 0 {
		pterm.DefaultLogger.Info(fmt.Sprintf("IP denylist active: %d CIDR(s)", len(IPDenylist)))
	}
}

func parseFirewallDuration(val string, defaultDur time.Duration) time.Duration {
	if val == "" {
		return defaultDur
	}
	d, err := time.ParseDuration(val)
	if err != nil || d <= 0 {
		return defaultDur
	}
	return d
}

func parseFirewallMax(val string, defaultMax int) int {
	if val == "" {
		return defaultMax
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return defaultMax
	}
	return n
}

func parseFirewallCIDRList(val string) []string {
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	var result []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
