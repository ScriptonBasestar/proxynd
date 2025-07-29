package app

import (
	"sync"

	"proxynd/logging"
)

// ConfigChangeNotifier 설정 변경 알림 시스템
type ConfigChangeNotifier struct {
	listeners []func(interface{})
	mu        sync.RWMutex
	logger    logging.Logger
}

// NewConfigChangeNotifier 새로운 설정 변경 알림자 생성
func NewConfigChangeNotifier() *ConfigChangeNotifier {
	return &ConfigChangeNotifier{
		listeners: make([]func(interface{}), 0),
		logger:    logging.GetLogger(),
	}
}

// AddListener 설정 변경 시 호출될 리스너 추가
func (n *ConfigChangeNotifier) AddListener(listener func(interface{})) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.listeners = append(n.listeners, listener)
	n.logger.Info("Config change listener added", logging.F("total_listeners", len(n.listeners)))
}

// RemoveListener 특정 리스너 제거 (실제로는 구현이 복잡하므로 Clear 사용 권장)
func (n *ConfigChangeNotifier) RemoveListener(_ func(interface{})) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 함수 포인터 비교는 어려우므로 전체 리스너 목록을 필터링하지 않음
	// 대신 Clear 메서드를 사용하여 모든 리스너를 제거하는 것을 권장
	n.logger.Warn("RemoveListener는 구현되지 않음. ClearListeners 사용 권장")
}

// ClearListeners 모든 리스너 제거
func (n *ConfigChangeNotifier) ClearListeners() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.listeners = make([]func(interface{}), 0)
	n.logger.Info("All config change listeners cleared")
}

// NotifyChange 설정 변경을 모든 리스너에 알림
func (n *ConfigChangeNotifier) NotifyChange(config interface{}) {
	n.mu.RLock()
	listeners := make([]func(interface{}), len(n.listeners))
	copy(listeners, n.listeners)
	n.mu.RUnlock()

	n.logger.Info("Notifying config change", logging.F("listener_count", len(listeners)))

	// 각 리스너를 고루틴에서 실행하여 블로킹 방지
	for i, listener := range listeners {
		go func(idx int, l func(interface{})) {
			defer func() {
				if r := recover(); r != nil {
					n.logger.Error("Config change listener panicked",
						logging.F("listener_index", idx),
						logging.F("panic", r))
				}
			}()

			l(config)
		}(i, listener)
	}
}

// GetListenerCount 등록된 리스너 수 반환
func (n *ConfigChangeNotifier) GetListenerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.listeners)
}
