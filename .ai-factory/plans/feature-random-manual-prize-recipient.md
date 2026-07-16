# План реализации: второе случайное назначение ручного приза

Ветка: `feature/random-manual-prize-recipient`
Создан: 2026-07-16

## Цель

Добавить рядом с существующим сценарием «без награды» второе действие «Отдать рандомному участнику» в админке и Mini App. Новое действие выбирает случайного допустимого участника активного события, даже если у него уже есть автоматический или вручную назначенный приз.

## Настройки

- Тестирование: да — добавить backend и frontend регрессионное покрытие для обоих независимых сценариев.
- Логирование: verbose — использовать текущий стиль `log.Printf` с префиксами `DEBUG`/`INFO`/`WARN`/`ERROR` и безопасные `console`-логи на frontend. Глобальный `LOG_LEVEL` не добавлять: его в проекте нет.
- Документация: да — в конце обязателен checkpoint через `$aif-docs`.

## Границы и решения

- Существующие кнопки и endpoints `POST /api/gifts/{id}/random-recipient` и `POST /api/miniapp/my-gifts/{giftId}/random-recipient` остаются без изменений: они выбирают только участника без автоматического и ручного приза.
- Новое действие доступно только для проверенного ручного приза без назначенного получателя. В админке старая кнопка сохраняет текущую доступность и для автоматического нераспределённого приза; новая кнопка не превращает автоматический приз в ручной.
- Новый пул исключает только запрет `has_prize`; остальные текущие правила допустимости сохраняются: участник того же события должен финишировать или иметь DNF и не быть дисквалифицированным. DNS/участник без результата и дисквалифицированный не выбираются.
- Выбор остаётся серверным и криптографически случайным. Клиент не передаёт список, ID получателя или флаг, меняющий семантику старого endpoint.
- Mini App сохраняет проверку Telegram init data, ownership подарка и active-event scope; чужой подарок по-прежнему не раскрывается. Admin API остаётся за существующим JWT admin middleware.
- Получатель не попадает в публичные `GiftDTO`, каталог или automatic prize-distribution read model. Миграция не нужна: ручному получателю уже разрешено иметь несколько призов.

## Commit Plan

- **Commit 1** (после задач 1-3): `feat(gifts): add random manual recipient assignment`
- **Commit 2** (после задач 4-6): `feat(ui): add random recipient actions`

## Задачи

### Этап 1: серверный выбор и атомарное назначение

- [x] Задача 1: Добавить отдельный query и Mini App command для случайного допустимого получателя с уже имеющимся призом.
  - Файлы:
    - `backend/internal/application/query/get_eligible_manual_gift_participant_ids.go`
    - `backend/internal/application/query/get_eligible_manual_gift_participant_ids_test.go`
    - `backend/internal/application/command/assign_random_manual_gift_recipient.go` или новый command-файл рядом с ним
    - `backend/internal/application/command/assign_random_manual_gift_recipient_test.go`
  - Результат:
    - Создать независимый query, который получает участников события через `ParticipantRepository.FindByEvent`, оставляет только `Participant.IsEligibleForManualGift()` и не читает automatic distribution или manual recipient counts.
    - Не менять `GetEligibleUnawardedParticipantIDsHandler`: старый query остаётся единственным источником для сценария «без награды».
    - Добавить отдельный owner-scoped command для нового Mini App действия: до выбора он вызывает существующую проверку owner/event/manual mode, использует `crypto/rand`, затем назначает результат через `SetManualGiftRecipientHandler`.
    - Ввести отдельную typed error для пустого списка допустимых участников; существующая `ErrManualGiftNoUnawardedParticipants` остаётся контрактом старой кнопки.
    - Покрыть финишировавшего/DNF с automatic и manual призами как допустимых, а DNS, DQ, `nil` и ошибку репозитория как исключения; проверить owner/event/manual-mode guard, пустой список и ошибку secure random.
  - Логирование:
    - `DEBUG` начало query/command и число кандидатов без имён, описаний призов или Telegram init data.
    - `INFO` успешное назначение с `gift_id`, `event_id`, количеством кандидатов и ID получателя.
    - `WARN` для пустого списка и нарушенных owner/event/manual инвариантов; `ERROR` для repository/crypto ошибок.
  - Зависимости: нет.

- [x] Задача 2: Реализовать для админки атомарное случайное назначение уже награждённому участнику, не ослабляя старую защиту «без награды».
  - Файлы:
    - `backend/internal/domain/repository/gift.go`
    - `backend/internal/application/command/assign_random_admin_gift_recipient.go` или новый command-файл рядом с ним
    - `backend/internal/application/command/assign_random_admin_gift_recipient_test.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo.go`
    - `backend/internal/infrastructure/persistence/postgres/gift_repo_test.go`
  - Результат:
    - Добавить узкий repository interface для compare-and-set записи, которая допускает прежние automatic/manual награды у кандидата; не расширять базовый `GiftRepository`.
    - Сохранить существующий `AssignRandomManualRecipient` и его проверку отсутствия наград для старого пути.
    - Новый admin command принимает только approved `manual_distribution=true` подарок без получателя, выбирает кандидата из задачи 1, дополнительно валидирует его событие и актуальную eligibility, затем вызывает новую транзакционную запись.
    - В PostgreSQL заблокировать целевой подарок и участника, повторно проверить approved/manual/unassigned состояние подарка и событие, а eligibility участника вычислить в той же транзакции по `participants.status` и наличию current `results` записи: допустимы finished или DNF, DNS/участник без current result и DQ отклоняются. Не полагаться только на pre-transaction проверку command.
    - Обновить только `manual_recipient_participant_id` и намеренно не проверять наличие других призов. Гонка на одном подарке должна вернуть конфликт, а не перезаписать получателя.
    - Не конвертировать автоматический приз и не инвалидировать публичный Mini App catalog cache: новое действие работает только с уже ручным призом.
    - Покрыть success с ранее automatic и manual наградой у кандидата, pending/automatic/already-assigned gift, пустой список, cross-event/DNS/DQ, потерю current result или смену статуса между выбором и записью, crypto failure и compare-and-set race.
  - Логирование:
    - `DEBUG` этапы выбора, транзакции и CAS только с техническими ID.
    - `INFO` успешное admin-назначение с `admin_id`, `gift_id`, `event_id`, `recipient_participant_id` и количеством кандидатов.
    - `WARN` для невалидного статуса, конфликта, гонки и отсутствия кандидатов; `ERROR` для SQL/transaction ошибок. Не логировать JWT, тела запросов и персональные поля.
  - Зависимости: задача 1.

- [x] Задача 3: Подключить новые защищённые HTTP endpoints, сохранив отдельную семантику и OpenAPI-контракт старого маршрута.
  - Файлы:
    - `backend/internal/infrastructure/http/server.go`
    - `backend/internal/infrastructure/http/handler/gifts.go`
    - `backend/internal/infrastructure/http/handler/gifts_random_recipient_test.go`
    - `backend/internal/infrastructure/http/handler/miniapp.go`
    - `backend/internal/infrastructure/http/handler/miniapp_test.go`
    - `backend/docs/swagger.yaml`
  - Результат:
    - Добавить `POST /api/gifts/{id}/random-recipient-including-awarded` под действующей admin авторизацией и `POST /api/miniapp/my-gifts/{giftId}/random-recipient-including-awarded` под Telegram init-data middleware.
    - В HTTP handlers явно выделить вторые поля и interfaces для wide-random command: отдельные dependency/constructor parameter в `GiftsHandler`, отдельное поле и parameter `ConfigureManualGiftManagement` в `MiniappHandler`, а также отдельные fakes в HTTP tests. Это не даёт новому маршруту по ошибке вызвать старый строгий selector.
    - Новые маршруты возвращают `204 No Content`, `404` для отсутствующего или скрытого по owner/event подарка, `409` для неготового/не ручного/уже назначенного приза или пустого пула и `500` только для непредвиденных сбоев.
    - Сохранить current 404 anti-enumeration и active-event lookup Mini App, а также cache-инвалидацию только у старого пути, который может перевести automatic gift в ручной режим.
    - Дополнить Swagger двумя маршрутами и описать точную разницу: `random-recipient` — только без наград, `random-recipient-including-awarded` — награждённые допустимы; оба без request body и с теми же eligibility/authorization границами.
    - Добавить handler tests на routing, 204, error mapping, old endpoint regression, отдельный wide-random fake и отсутствие конфликта между handler dependencies.
  - Логирование:
    - `DEBUG` decoded path ID и безопасный execution stage.
    - `INFO` только успешные назначения с actor type и ID; `WARN` для 400/404/409; `ERROR` для wiring/command failures.
    - Не логировать Telegram init data, JWT, имена, usernames или описания призов.
  - Зависимости: задачи 1-2.

### Этап 2: две понятные пользовательские операции

- [x] Задача 4: Добавить в админке отдельную кнопку и API-клиент для ручного random-назначения участнику, который уже может иметь приз.
  - Файлы:
    - `frontend/src/api/gifts.ts`
    - `frontend/src/api/gifts.test.ts`
    - `frontend/src/utils/manualGiftAssignment.ts`
    - `frontend/src/utils/manualGiftAssignment.test.ts`
    - `frontend/src/components/gifts/GiftsTable.tsx`
    - `frontend/src/components/gifts/GiftsTable.test.tsx` (создать)
    - `frontend/src/app/(dashboard)/gifts/page.tsx`
    - `frontend/src/app/(dashboard)/gifts/page.test.tsx`
  - Результат:
    - Добавить typed `giftsApi.assignRandomRecipientIncludingAwarded(id)` с отдельным POST URL и пустым телом; существующий клиент не менять.
    - Добавить чистый predicate, разрешающий новую кнопку только для approved `manual_unassigned` подарка, и оставить `canAssignRandomRecipient` без изменения.
    - Отобразить рядом с текущим действием точную кнопку «Отдать рандомному участнику», с понятным tooltip/accessible name; сохранить текущую кнопку и её значение «без приза».
    - Передать отдельный callback из страницы, после успеха обновлять список тем же source-of-truth flow, а во время любого random запроса для одной строки блокировать обе random-кнопки и исключить двойную мутацию.
    - Протестировать URL/метод/пустое body, видимость только для ручного нераспределённого приза, обе подписи и блокировку, reload после успеха и ошибку без обновления UI.
  - Логирование:
    - Использовать безопасные `console.info`/`console.error` с `operation`, `gift_id`, `event_id` и ошибкой без описаний подарка, имён или usernames.
    - Не добавлять логи в чистые presentation utilities.
  - Зависимости: задача 3.

- [x] Задача 5: Добавить в Mini App вторую кнопку, API-вызов и корректные action-specific ошибки.
  - Файлы:
    - `frontend/src/api/miniapp.ts`
    - `frontend/src/api/miniapp.test.ts`
    - `frontend/src/utils/miniappMyGifts.ts`
    - `frontend/src/utils/miniappMyGifts.test.ts`
    - `frontend/src/components/miniapp/MyGiftRecipientSelect.tsx`
    - `frontend/src/components/miniapp/MyGiftRecipientSelect.test.tsx` (создать)
    - `frontend/src/components/miniapp/MyGiftCard.tsx`
    - `frontend/src/components/miniapp/MyGiftCard.test.tsx`
    - `frontend/src/app/(miniapp)/miniapp/my-gifts/page.tsx`
    - `frontend/src/app/(miniapp)/miniapp/my-gifts/page.test.tsx`
  - Результат:
    - Добавить `miniappApi.assignRandomMyGiftRecipientIncludingAwarded(giftID)` и отдельный callback через page → card → recipient selector.
    - Для ручного приза без получателя отобразить рядом обе понятные кнопки: текущую «Отдать рандомному участнику без награды» и новую точную «Отдать рандомному участнику»; предусмотреть мобильное размещение без сжатия текста.
    - Использовать существующий `savingGiftID`: обе кнопки и обычный выбор получателя блокируются на время любой мутации и последующего refresh.
    - После `204` заново загрузить защищённый `/my-gifts`; при сбое refresh не скрывать факт успешного назначения и сохранить `MiniappMyGiftsRefreshError` flow.
    - Добавить общий для `load`, обычного ручного выбора и обоих random actions mount/version guard: cleanup инвалидацирует активный запрос, поздний response не обновляет page state или snapshot provider, а `finally` не вызывает `setSavingGiftID` после unmount. При открытой странице сохраняется current source-of-truth refresh flow.
    - Сделать 409 copy зависящим от действия или нейтральным, чтобы новый путь не сообщал ошибочно, что нужен участник «без награды».
    - Покрыть отдельный API URL, обе callbacks/подписи, доступность только до назначения получателя, saving state, action-specific 409, successful refresh, refresh failure, unmount во время mutation и поздний refresh, который не должен перезаписать актуальный snapshot.
  - Логирование:
    - `console.info` успешного действия и `console.warn` ошибки включают только `operation`, `gift_id` и безопасное техническое сообщение.
    - Не передавать в консоль participant options, Telegram init data, имя или username получателя.
  - Зависимости: задача 3.

### Этап 3: документация и проверка

- [x] Задача 6: Провести обязательный documentation checkpoint и подтвердить независимость двух сценариев полным набором проверок.
  - Файлы:
    - `README.md` (только существующий раздел Mini App/ручных призов, если он требует обновления)
    - `backend/docs/swagger.yaml` (финальная сверка примеров и ошибок)
    - только implementation/test-файлы, исправления в которых выявит проверка
  - Результат:
    - Выполнить `$aif-docs` checkpoint: документировать пользовательское различие двух кнопок и не создавать отдельные отчёты или Markdown-инструкции. В существующем README исправить неверное утверждение о получателе ручного приза: оно должно соответствовать текущему правилу — finished или DNF допустимы, участник без current result/DNS и DQ исключаются; не расширять ради документации продуктовую eligibility-логику.
    - Проверить, что Swagger, API clients и маршруты используют одинаковые отдельные URL и что старые consumers не изменили поведение.
    - Выполнить целевые Go и Vitest тесты в ходе работы, затем `GOCACHE=/private/tmp/gravel_bot-go-build go test ./...` из `backend`, `npm test -- --run`, `npm run build` и `npm run lint` из `frontend`; известный baseline lint отделить от новой регрессии.
    - Запустить релевантный Docker Compose runtime и проверить обе поверхности: старая кнопка не выбирает уже награждённого, новая может выбрать участника с automatic и с manual призом, но не DNS/DQ; Mini App сохраняет owner/event privacy, а admin не может вызвать новую операцию для automatic приза.
    - Завершить `git diff --check` и просмотром полного branch diff относительно `origin/main`.
  - Логирование:
    - Подтвердить наличие planned `DEBUG`/`INFO`/`WARN`/`ERROR` границ на backend и отсутствие чувствительных данных в логах.
    - Не добавлять runtime-логи в документацию или тестовые фикстуры с реальными Telegram/JWT данными.
  - Зависимости: задачи 1-5.
