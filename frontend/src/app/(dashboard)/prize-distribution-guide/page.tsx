import Link from 'next/link';

const priorityRows = [
  {
    level: '1',
    name: 'Критерии + место',
    meaning: 'Приз сначала фильтрует участников по критериям результата, затем выбирает место внутри этой группы.',
    example: 'Например: критерий "самый высокий пульс" + место 1.',
  },
  {
    level: '2',
    name: 'Критерии без места',
    meaning: 'Приз получает первый подходящий участник, у результата которого есть все критерии подарка.',
    example: 'Например: "Набрал больше ЦГ".',
  },
  {
    level: '3',
    name: 'Место без критериев',
    meaning: 'Приз получает участник на указанном месте в подходящей группе.',
    example: 'Например: "Лабуба за 1 место".',
  },
  {
    level: '4',
    name: 'Без критериев и места',
    meaning: 'Обычный общий приз. Он уходит первому свободному подходящему участнику.',
    example: 'Например: рандомный приз без условий.',
  },
];

const quickRules = [
  {
    title: 'Участвуют только одобренные призы',
    body: 'Подарок со статусом "На проверке" не матчится ни с кем.',
  },
  {
    title: 'Критерии сужают группу',
    body: 'Если у приза есть критерии, сначала остаются только результаты с этими критериями.',
  },
  {
    title: 'Место считается внутри группы',
    body: 'Место 1 может быть первым среди мужчин, гравийников или результатов с нужным критерием.',
  },
  {
    title: 'Приз за место не пропадает',
    body: 'Участник может получить и критерийный приз, и отдельный слот за своё место.',
  },
];

const matchingSteps = [
  'Берутся только подарки со статусом "Одобрен". Подарки на проверке не участвуют в распределении.',
  'Для каждого подарка собирается подходящая группа участников: пол, тип велосипеда и критерии результата.',
  'Места считаются заново внутри подходящей группы, а не всегда по абсолютному месту общего протокола.',
  'Если у подарка есть несколько мест, одно описание подарка разворачивается в несколько слотов.',
  'Слоты с местами могут выдаваться участнику, даже если он уже получил приз по критерию.',
  'Обычные призы без места не добиваются к участнику, который уже занят более приоритетным критериальным призом.',
];

const ruleCards = [
  {
    title: 'places',
    tag: 'места 1, 3, 10-15',
    body: 'Выдаёт слот каждому указанному месту. Диапазоны нормализуются, дубли удаляются, порядок становится возрастающим.',
  },
  {
    title: 'last_n',
    tag: '2 последних',
    body: 'Берёт N последних участников внутри уже отфильтрованной группы. Для last_n fallback не используется: слот либо попал в группу, либо группы нет.',
  },
  {
    title: 'legacy place',
    tag: 'старое поле place',
    body: 'Старое поле места работает как places с одним значением. Если есть новый place_rule, он важнее legacy place.',
  },
  {
    title: 'none',
    tag: 'без привязки',
    body: 'Приз без места выдаётся первому подходящему участнику, которого ещё не занял более приоритетный no-place уровень.',
  },
];

const exampleRows = [
  {
    case: 'Участник занимает 1 место и у результата есть выбранный критерий',
    result: 'Он получает и критерийный приз, и приз за 1 место. Место не пропадает из-за критерия.',
  },
  {
    case: 'Приз: критерий "Набрал больше ЦГ" + places: 1',
    result: 'Сначала остаются только результаты с этим критерием. Место 1 считается среди них.',
  },
  {
    case: 'Приз: gender=male, bike=gravel, places: 1-3',
    result: 'Слоты 1, 2 и 3 считаются только среди мужчин на гравийниках.',
  },
  {
    case: 'Приз: places: 10, но в группе только 4 участника',
    result: 'Движок ищет ближайшего свободного участника. Сначала ниже целевого места, затем выше.',
  },
  {
    case: 'Два общих приза без места и первый участник уже получил criteria-приз',
    result: 'Общие призы уйдут следующему свободному участнику, а не добавятся к занятому первому.',
  },
];

const diagnostics = [
  {
    label: 'target_rank',
    text: 'какое место просил слот подарка',
  },
  {
    label: 'assigned_rank',
    text: 'какому месту слот реально выдан после фильтров и fallback',
  },
  {
    label: 'is_fallback',
    text: 'true, если целевое место было недоступно и выбран ближайший свободный участник',
  },
  {
    label: 'unassigned_slots',
    text: 'слоты, для которых не нашлось ни одного подходящего участника',
  },
];

function SectionTitle({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description: string;
}) {
  return (
    <div className="max-w-3xl">
      <p className="text-xs font-semibold uppercase text-brand-500">
        {eyebrow}
      </p>
      <h2 className="mt-2 text-xl font-semibold text-gray-900 dark:text-white">
        {title}
      </h2>
      <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
        {description}
      </p>
    </div>
  );
}

export default function PrizeDistributionGuidePage() {
  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:flex-row lg:items-start lg:justify-between lg:p-6">
        <div className="max-w-4xl">
          <p className="text-sm font-medium text-brand-500">Справочник модератора</p>
          <h1 className="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            Как распределяются призы
          </h1>
          <p className="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
            Короткая памятка по матчингу подарков с результатами. Сначала смотрите базовые
            правила ниже, подробные случаи разобраны дальше на странице.
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          <Link
            href="/prize-distribution"
            className="inline-flex items-center justify-center rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 transition hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-white/[0.04]"
          >
            Открыть распределение
          </Link>
          <Link
            href="/gifts"
            className="inline-flex items-center justify-center rounded-lg bg-brand-500 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-brand-600"
          >
            Проверить призы
          </Link>
        </div>
      </div>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
        <SectionTitle
          eyebrow="Коротко"
          title="Главные правила"
          description="Этого достаточно для быстрой проверки большинства вопросов."
        />

        <div className="mt-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          {quickRules.map((rule) => (
            <div key={rule.title} className="rounded-lg border border-gray-200 p-4 dark:border-gray-800">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
                {rule.title}
              </h3>
              <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {rule.body}
              </p>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
        <SectionTitle
          eyebrow="Алгоритм"
          title="Порядок расчёта"
          description="Подробная последовательность для случаев, когда нужно понять конкретную выдачу."
        />

        <ol className="mt-6 space-y-3">
          {matchingSteps.map((step, index) => (
            <li key={step} className="flex gap-3">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-brand-50 text-sm font-semibold text-brand-600 dark:bg-brand-500/15 dark:text-brand-300">
                {index + 1}
              </span>
              <p className="pt-1 text-sm leading-6 text-gray-700 dark:text-gray-300">
                {step}
              </p>
            </li>
          ))}
        </ol>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
          <SectionTitle
            eyebrow="Важное"
            title="Что чаще всего путают"
            description="Если приз кажется неверным, сначала проверьте эти правила."
          />

          <div className="mt-6 space-y-3">
            <div className="rounded-lg border border-success-200 bg-success-50 p-4 dark:border-success-900/60 dark:bg-success-900/15">
              <p className="text-sm font-semibold text-success-700 dark:text-success-300">
                Приз за место и приз за критерий могут быть у одного участника
              </p>
              <p className="mt-2 text-sm leading-6 text-success-800/80 dark:text-success-200/80">
                Это ожидаемо для слотов с places или last_n. Место считается отдельным слотом.
              </p>
            </div>
            <div className="rounded-lg border border-warning-200 bg-warning-50 p-4 dark:border-warning-900/60 dark:bg-warning-900/15">
              <p className="text-sm font-semibold text-warning-700 dark:text-warning-300">
                Критерии фильтруют группу до расчёта места
              </p>
              <p className="mt-2 text-sm leading-6 text-warning-800/80 dark:text-warning-200/80">
                Место 1 у критерийного подарка означает первое место среди результатов с этим критерием.
              </p>
            </div>
            <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/40">
              <p className="text-sm font-semibold text-gray-800 dark:text-white">
                На проверке не распределяется
              </p>
              <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
                Пока модератор не одобрил подарок, он не участвует ни в одной очереди.
              </p>
            </div>
          </div>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
        <SectionTitle
          eyebrow="Приоритет"
          title="Какие призы матчятся первыми"
          description="Приоритет нужен для no-place призов: он не даёт общим подаркам забрать участника, который уже выиграл более точный критерийный приз."
        />

        <div className="mt-6 overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-800">
          <div className="min-w-[860px]">
            <div className="grid grid-cols-[72px_minmax(160px,0.8fr)_minmax(220px,1.2fr)_minmax(180px,0.9fr)] bg-gray-50 text-xs font-semibold uppercase text-gray-500 dark:bg-gray-900/60 dark:text-gray-400">
              <div className="px-4 py-3">#</div>
              <div className="px-4 py-3">Тип</div>
              <div className="px-4 py-3">Как работает</div>
              <div className="px-4 py-3">Пример</div>
            </div>
            {priorityRows.map((row) => (
              <div
                key={row.level}
                className="grid grid-cols-[72px_minmax(160px,0.8fr)_minmax(220px,1.2fr)_minmax(180px,0.9fr)] border-t border-gray-200 text-sm dark:border-gray-800"
              >
                <div className="px-4 py-4 font-semibold text-brand-600 dark:text-brand-300">
                  {row.level}
                </div>
                <div className="px-4 py-4 font-medium text-gray-900 dark:text-white">
                  {row.name}
                </div>
                <div className="px-4 py-4 leading-6 text-gray-600 dark:text-gray-400">
                  {row.meaning}
                </div>
                <div className="px-4 py-4 leading-6 text-gray-600 dark:text-gray-400">
                  {row.example}
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
        <SectionTitle
          eyebrow="Правила мест"
          title="Как читать place_rule"
          description="Один подарок может создать несколько выдаваемых слотов. Это не дублирование подарка, а разные позиции одного правила."
        />

        <div className="mt-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          {ruleCards.map((rule) => (
            <div key={rule.title} className="rounded-lg border border-gray-200 p-4 dark:border-gray-800">
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-base font-semibold text-gray-900 dark:text-white">
                  {rule.title}
                </h3>
                <span className="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-gray-800 dark:text-gray-300">
                  {rule.tag}
                </span>
              </div>
              <p className="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {rule.body}
              </p>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
        <SectionTitle
          eyebrow="Примеры"
          title="Как разбирать спорные случаи"
          description="Эти сценарии покрывают большинство вопросов, которые возникают у модераторов при ручной проверке."
        />

        <div className="mt-6 divide-y divide-gray-200 rounded-lg border border-gray-200 dark:divide-gray-800 dark:border-gray-800">
          {exampleRows.map((row) => (
            <div key={row.case} className="grid grid-cols-1 gap-3 p-4 md:grid-cols-[minmax(220px,0.9fr)_minmax(260px,1.1fr)]">
              <p className="text-sm font-medium leading-6 text-gray-900 dark:text-white">
                {row.case}
              </p>
              <p className="text-sm leading-6 text-gray-600 dark:text-gray-400">
                {row.result}
              </p>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
        <SectionTitle
          eyebrow="Диагностика"
          title="Что означают поля в распределении"
          description="Эти значения показываются в API и используются на странице распределения призов."
        />

        <div className="mt-6 grid grid-cols-1 gap-3 md:grid-cols-2">
          {diagnostics.map((item) => (
            <div key={item.label} className="rounded-lg border border-gray-200 p-4 dark:border-gray-800">
              <code className="rounded bg-gray-100 px-2 py-1 text-sm font-semibold text-gray-800 dark:bg-gray-800 dark:text-gray-200">
                {item.label}
              </code>
              <p className="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-400">
                {item.text}
              </p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
