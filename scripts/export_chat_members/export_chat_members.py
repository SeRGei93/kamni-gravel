#!/usr/bin/env python3
"""Экспорт участников Telegram-чата в CSV через MTProto (Telethon).

Bot API не умеет отдавать список членов чата, поэтому выгрузка делается
от имени живого аккаунта, состоящего в чате (для полного списка лучше админ).

Подготовка:
  1. Получите api_id и api_hash на https://my.telegram.org -> API development tools.
  2. pip install -r requirements.txt

Запуск:
  TG_API_ID=12345 TG_API_HASH=abcdef... \
    python export_chat_members.py --chat -1001234567890

  --chat принимает ID в формате Bot API (-100...), @username или ссылку на чат.
  Если --chat не указан, берётся PUBLIC_CHAT_ID из окружения.

При первом запуске Telethon спросит номер телефона и код подтверждения,
после чего сохранит сессию в файл *.session рядом со скриптом — повторный
логин не потребуется. Файл сессии содержит ключи доступа к аккаунту:
не коммитьте его и не передавайте никому.

Ограничение Telegram: для очень больших супергрупп (> ~10 000 участников)
сервер может отдать не всех. Скрипт в конце сверяет число выгруженных строк
с общим числом участников и предупреждает о расхождении.
"""

import argparse
import asyncio
import csv
import os
import sys

from telethon import TelegramClient
from telethon.tl.types import (
    ChannelParticipantAdmin,
    ChannelParticipantCreator,
)

CSV_COLUMNS = [
    "user_id",
    "username",
    "first_name",
    "last_name",
    "is_bot",
    "is_deleted",
    "role",
    "joined_at",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Выгрузка участников чата в CSV")
    parser.add_argument(
        "--chat",
        default=os.environ.get("PUBLIC_CHAT_ID"),
        help="ID чата (-100...), @username или ссылка; по умолчанию $PUBLIC_CHAT_ID",
    )
    parser.add_argument(
        "--output",
        default="chat_members.csv",
        help="путь к выходному CSV (по умолчанию chat_members.csv)",
    )
    parser.add_argument(
        "--session",
        default="chat_export",
        help="имя файла сессии Telethon (по умолчанию chat_export.session)",
    )
    args = parser.parse_args()

    if not args.chat:
        parser.error("не задан чат: укажите --chat или переменную окружения PUBLIC_CHAT_ID")
    return args


def require_env(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        sys.exit(f"Не задана переменная окружения {name} (см. https://my.telegram.org)")
    return value


def normalize_chat(raw: str):
    """Числовой ID приводим к int (Telethon понимает формат -100...), остальное — как есть."""
    raw = raw.strip()
    try:
        return int(raw)
    except ValueError:
        return raw


def participant_role(user) -> str:
    participant = getattr(user, "participant", None)
    if isinstance(participant, ChannelParticipantCreator):
        return "creator"
    if isinstance(participant, ChannelParticipantAdmin):
        return "admin"
    return "member"


def participant_joined_at(user) -> str:
    date = getattr(getattr(user, "participant", None), "date", None)
    return date.isoformat() if date else ""


async def export(chat, output_path: str, session: str, api_id: int, api_hash: str) -> None:
    client = TelegramClient(session, api_id, api_hash)
    await client.start()
    async with client:
        entity = await client.get_entity(chat)
        title = getattr(entity, "title", str(chat))
        print(f"Чат: {title}")

        total = (await client.get_participants(entity, limit=0)).total
        print(f"Участников по данным Telegram: {total}")

        exported = 0
        with open(output_path, "w", newline="", encoding="utf-8-sig") as f:
            writer = csv.writer(f)
            writer.writerow(CSV_COLUMNS)
            async for user in client.iter_participants(entity):
                writer.writerow(
                    [
                        user.id,
                        user.username or "",
                        user.first_name or "",
                        user.last_name or "",
                        int(bool(user.bot)),
                        int(bool(user.deleted)),
                        participant_role(user),
                        participant_joined_at(user),
                    ]
                )
                exported += 1

        print(f"Экспортировано строк: {exported} -> {output_path}")
        if exported < total:
            print(
                f"ВНИМАНИЕ: выгружено меньше, чем участников в чате ({exported} < {total}). "
                "Для больших супергрупп Telegram отдаёт не всех; попробуйте запустить "
                "от аккаунта с правами администратора."
            )


def main() -> None:
    args = parse_args()
    api_id = int(require_env("TG_API_ID"))
    api_hash = require_env("TG_API_HASH")
    asyncio.run(export(normalize_chat(args.chat), args.output, args.session, api_id, api_hash))


if __name__ == "__main__":
    main()
