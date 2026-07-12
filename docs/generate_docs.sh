#!/bin/bash
mkdir -p .context

# 1. Структура каталогов
tree -I 'vendor|.git|bin|tmp|coverage.out' --dirsfirst > .context/tree.txt

# 2. Документация пакетов
for pkg in $(go list ./...); do
  echo "=== $pkg ===" >> .context/project_docs.txt
  go doc -all "$pkg" >> .context/project_docs.txt
  echo "" >> .context/project_docs.txt
done

# 3. Ключевые файлы
cp Payship_Core_architecture.drawio.xml .context/ 2>/dev/null
cp go.mod docker-compose.yml README.md .context/ 2>/dev/null

echo "DONE: .context.tar.gz"