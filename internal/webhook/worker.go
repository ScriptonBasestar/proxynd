package webhook

import (
	"context"
	"time"

	"proxynd/alerts"
	"proxynd/logging"
)

// Start 워커 시작
func (w *Worker) Start(ctx context.Context) {
	defer w.sender.wg.Done()

	w.logger.Info("웹훅 워커 시작됨", logging.F("worker_id", w.id))

	for {
		select {
		case event := <-w.eventCh:
			w.processEvent(ctx, event)
		case <-w.stopCh:
			w.logger.Info("웹훅 워커 중지됨", logging.F("worker_id", w.id))
			return
		case <-ctx.Done():
			w.logger.Info("웹훅 워커 컨텍스트 종료", logging.F("worker_id", w.id))
			return
		}
	}
}

// processEvent 이벤트 처리
func (w *Worker) processEvent(ctx context.Context, event *alerts.AlertEvent) {
	w.logger.Debug("이벤트 처리 시작",
		logging.F("worker_id", w.id),
		logging.F("event_id", event.ID),
		logging.F("event_type", event.Type))

	// 모든 활성화된 엔드포인트로 전송
	for _, endpoint := range w.sender.config.Endpoints {
		if !endpoint.Enabled {
			continue
		}

		// 엔드포인트별 필터 확인
		if !w.sender.matchesEndpointFilter(event, endpoint) {
			w.logger.Debug("엔드포인트 필터로 제외됨",
				logging.F("endpoint", endpoint.Name),
				logging.F("event_type", event.Type))
			continue
		}

		// 전송 시도
		if err := w.sender.sendToEndpoint(ctx, event, endpoint); err != nil {
			w.logger.Error("엔드포인트 전송 실패",
				logging.F("worker_id", w.id),
				logging.F("endpoint", endpoint.Name),
				logging.F("event_id", event.ID),
				logging.F("error", err.Error()))

			// 재시도 큐에 추가 (심각한 오류가 아닌 경우)
			if w.sender.isRetryableError(err) {
				retryItem := &RetryItem{
					Event:     event,
					Endpoint:  endpoint.Name,
					Attempt:   1,
					NextRetry: time.Now().Add(time.Second * 5),
					LastError: err,
					CreatedAt: time.Now(),
				}
				w.retryQ.Add(retryItem)
			}
		} else {
			w.logger.Debug("엔드포인트 전송 성공",
				logging.F("worker_id", w.id),
				logging.F("endpoint", endpoint.Name),
				logging.F("event_id", event.ID))
		}
	}
}

// NewRetryQueue 새로운 재시도 큐 생성
func NewRetryQueue() *RetryQueue {
	return &RetryQueue{
		items:    make([]*RetryItem, 0),
		notifyCh: make(chan struct{}, 1),
	}
}

// Add 재시도 항목 추가
func (rq *RetryQueue) Add(item *RetryItem) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.items = append(rq.items, item)

	// 알림 전송 (non-blocking)
	select {
	case rq.notifyCh <- struct{}{}:
	default:
	}
}

// processRetries 재시도 처리
func (rq *RetryQueue) processRetries(ctx context.Context, sender *WebhookSender) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	now := time.Now()
	newItems := make([]*RetryItem, 0)

	for _, item := range rq.items {
		// 재시도 시간이 되었는지 확인
		if now.Before(item.NextRetry) {
			newItems = append(newItems, item)
			continue
		}

		// 최대 재시도 횟수 확인
		if item.Attempt >= sender.config.Retry.MaxAttempts {
			sender.logger.Warn("재시도 한도 초과로 포기",
				logging.F("event_id", item.Event.ID),
				logging.F("endpoint", item.Endpoint),
				logging.F("attempts", item.Attempt))

			// 실패 처리 (Dead Letter Queue 등)
			sender.handlePermanentFailure(item)
			continue
		}

		// 엔드포인트 찾기
		var endpoint *configs.WebhookEndpointConfig
		for _, ep := range sender.config.Endpoints {
			if ep.Name == item.Endpoint {
				endpoint = &ep
				break
			}
		}

		if endpoint == nil {
			sender.logger.Warn("엔드포인트를 찾을 수 없음",
				logging.F("endpoint", item.Endpoint))
			continue
		}

		// 재시도 전송
		if err := sender.sendToEndpoint(ctx, item.Event, *endpoint); err != nil {
			// 실패 시 재시도 항목 업데이트
			item.Attempt++
			item.LastError = err
			item.NextRetry = now.Add(sender.calculateBackoffDelay(item.Attempt, RetryPolicy{
				MaxAttempts:   sender.config.Retry.MaxAttempts,
				InitialDelay:  time.Second,
				MaxDelay:      time.Minute * 5,
				BackoffFactor: sender.config.Retry.BackoffFactor,
			}))

			newItems = append(newItems, item)

			sender.logger.Warn("재시도 실패",
				logging.F("event_id", item.Event.ID),
				logging.F("endpoint", item.Endpoint),
				logging.F("attempt", item.Attempt),
				logging.F("next_retry", item.NextRetry),
				logging.F("error", err.Error()))
		} else {
			// 성공
			sender.logger.Info("재시도 성공",
				logging.F("event_id", item.Event.ID),
				logging.F("endpoint", item.Endpoint),
				logging.F("attempt", item.Attempt))
		}
	}

	rq.items = newItems
}

// Size 재시도 큐 크기 반환
func (rq *RetryQueue) Size() int {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	return len(rq.items)
}

// Clear 재시도 큐 정리
func (rq *RetryQueue) Clear() {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	rq.items = rq.items[:0]
}

// handlePermanentFailure 영구 실패 처리
func (ws *WebhookSender) handlePermanentFailure(item *RetryItem) {
	if !ws.config.Retry.PersistFailures {
		return
	}

	// Dead Letter Queue에 저장 (파일 시스템 또는 별도 저장소)
	ws.logger.Error("영구 실패 이벤트 저장",
		logging.F("event_id", item.Event.ID),
		logging.F("endpoint", item.Endpoint),
		logging.F("total_attempts", item.Attempt),
		logging.F("error", item.LastError.Error()))

	// 실제 구현에서는 파일이나 데이터베이스에 저장
	// 여기서는 로그만 기록
}
