# ShiftNotifier
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