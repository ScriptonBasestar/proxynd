package factory

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/internal/adapters/http"
)

// Maven Adapter Wrapper
type mavenAdapterWrapper struct {
	adapter *http.MavenBrowserAdapter
}

func (w *mavenAdapterWrapper) Handle(c *fiber.Ctx) error {
	return w.adapter.Handle(c)
}

func (w *mavenAdapterWrapper) GetProxyType() string {
	return "maven"
}

func (w *mavenAdapterWrapper) HealthCheck() error {
	// 기본 헬스체크 구현
	return nil
}

func (w *mavenAdapterWrapper) Initialize() error {
	// 초기화 로직 (필요시 구현)
	return nil
}

func (w *mavenAdapterWrapper) Shutdown() error {
	// 정리 로직 (필요시 구현)
	return nil
}

// APT Adapter Wrapper
type aptAdapterWrapper struct {
	adapter *http.APTHandlerAdapter
}

func (w *aptAdapterWrapper) Handle(c *fiber.Ctx) error {
	return w.adapter.Handle(c)
}

func (w *aptAdapterWrapper) GetProxyType() string {
	return "apt"
}

func (w *aptAdapterWrapper) HealthCheck() error {
	return nil
}

func (w *aptAdapterWrapper) Initialize() error {
	return nil
}

func (w *aptAdapterWrapper) Shutdown() error {
	return nil
}

// NPM Adapter Wrapper
type npmAdapterWrapper struct {
	adapter *http.NPMHandlerAdapter
}

func (w *npmAdapterWrapper) Handle(c *fiber.Ctx) error {
	return w.adapter.Handle(c)
}

func (w *npmAdapterWrapper) GetProxyType() string {
	return "npm"
}

func (w *npmAdapterWrapper) HealthCheck() error {
	return nil
}

func (w *npmAdapterWrapper) Initialize() error {
	return nil
}

func (w *npmAdapterWrapper) Shutdown() error {
	return nil
}

// PIP Adapter Wrapper
type pipAdapterWrapper struct {
	adapter *http.PIPHandlerAdapter
}

func (w *pipAdapterWrapper) Handle(c *fiber.Ctx) error {
	return w.adapter.Handle(c)
}

func (w *pipAdapterWrapper) GetProxyType() string {
	return "pip"
}

func (w *pipAdapterWrapper) HealthCheck() error {
	return nil
}

func (w *pipAdapterWrapper) Initialize() error {
	return nil
}

func (w *pipAdapterWrapper) Shutdown() error {
	return nil
}

// Docker Adapter Wrapper
type dockerAdapterWrapper struct {
	adapter *http.DockerHandlerAdapter
}

func (w *dockerAdapterWrapper) Handle(c *fiber.Ctx) error {
	return w.adapter.Handle(c)
}

func (w *dockerAdapterWrapper) GetProxyType() string {
	return "docker"
}

func (w *dockerAdapterWrapper) HealthCheck() error {
	return nil
}

func (w *dockerAdapterWrapper) Initialize() error {
	return nil
}

func (w *dockerAdapterWrapper) Shutdown() error {
	return nil
}

// YUM Adapter Wrapper
type yumAdapterWrapper struct {
	adapter *http.YumHandlerAdapter
}

func (w *yumAdapterWrapper) Handle(c *fiber.Ctx) error {
	return w.adapter.Handle(c)
}

func (w *yumAdapterWrapper) GetProxyType() string {
	return "yum"
}

func (w *yumAdapterWrapper) HealthCheck() error {
	return nil
}

func (w *yumAdapterWrapper) Initialize() error {
	return nil
}

func (w *yumAdapterWrapper) Shutdown() error {
	return nil
}

// APK Adapter Wrapper
type apkAdapterWrapper struct {
	adapter *http.ApkHandlerAdapter
}

func (w *apkAdapterWrapper) Handle(c *fiber.Ctx) error {
	return w.adapter.Handle(c)
}

func (w *apkAdapterWrapper) GetProxyType() string {
	return "apk"
}

func (w *apkAdapterWrapper) HealthCheck() error {
	return nil
}

func (w *apkAdapterWrapper) Initialize() error {
	return nil
}

func (w *apkAdapterWrapper) Shutdown() error {
	return nil
}