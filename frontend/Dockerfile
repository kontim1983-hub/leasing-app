FROM node:16-alpine

WORKDIR /app

# Копируем package.json и устанавливаем зависимости
COPY package*.json ./
RUN npm install

# Копируем исходники
COPY . .

EXPOSE 3000

# Запускаем dev сервер
CMD ["npm", "start"]