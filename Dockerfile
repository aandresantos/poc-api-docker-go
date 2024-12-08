# Use a lightweight Node.js base image
FROM node:18-alpine

# Set the working directory
WORKDIR /app

# Copie o restante dos arquivos do projeto
COPY . .

# Exponha a porta padrão para Next.js
EXPOSE 3000

# Comando para rodar a aplicação
CMD ["touch", "teste.txt"]