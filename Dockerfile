# ---------- build stage ----------
FROM golang:1.25-alpine AS build
WORKDIR /src

# โหลด dependency ก่อน (cache layer — แก้โค้ดแล้วไม่ต้องโหลดใหม่)
COPY go.mod go.sum ./
RUN go mod download

# คอมไพล์เป็น static binary (CGO_ENABLED=0 → ไม่ต้องพึ่ง libc → ใส่ใน distroless ได้)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/catice ./cmd/catice

# ---------- run stage ----------
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /bin/catice /app/catice
# backend ตัวนี้เสิร์ฟแค่ API/WebSocket — frontend อยู่ที่ Vite (dev) หรือ static host (prod)
# main.go มี FileServer("web") อยู่ แต่ถ้าไม่มี web/ ก็แค่ตอบ 404 ที่ "/" (ไม่ crash)
# ถ้าจะให้ container เสิร์ฟ frontend เองแบบ standalone:
#   1) cd ../catice-frontend && npm run build   (output → Catice2/web)
#   2) เพิ่มบรรทัด: COPY web /app/web

EXPOSE 8080
ENTRYPOINT ["/app/catice"]
