# ShiftNotifier

## 概要

私のバイト先では、Excelを用いて講師のシフト管理を行っています。
ShiftNotifierは、そのExcelシフト表をLINE経由で取り込み、勤務開始前の講師へ自動で通知を送るために開発しているシステムです。

このシステムを運用することで、万が一シフトを忘れてしまった場合でも事前に気づくことができ、勤務忘れへの不安を軽減できます。
なお、本リポジトリで扱っている開発内容の掲載については、アルバイト先から許可を得ています。

起動方法
run を実行することで内部的に以下が同時に実行されます。
```
serve        // LINEからExcelを受け取る
notify-loop  // 5分ごとに通知対象を探してLINE通知
```
```
set -a
source .env
set +a

go run ./cmd/shift-notifier run
```

## .env で変更できる設定

```
SHIFT_NOTIFIER_REMINDER_BEFORE=2h      # 勤務開始の何時間/何分前に通知するか
SHIFT_NOTIFIER_NOTIFY_INTERVAL=5m      # 通知対象を探す間隔
```
