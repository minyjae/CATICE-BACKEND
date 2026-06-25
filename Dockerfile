# ---------- build stage ----------
FROM golang:1.25-alpine AS build
WORKDIR /src

# โหลด dependency ก่อน (cache layer — แก้โค้ดแล้วไม่ต้องโหลดใหม่)
COPY go.mod go.sum ./
RUN go mod download

# คอมไพล์เป็น static binary
#   CGO_ENABLED=0 → ไม่พึ่ง libc → ใส่ใน distroless/static ได้
#   -trimpath     → ตัด path เครื่อง build ออกจาก binary (reproducible + ไม่หลุด path เครื่อง)
#   -ldflags "-s -w" → ตัด symbol/debug table → binary เล็กลง
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/catice ./cmd/catice

# ---------- run stage ----------
# distroless/static = ไม่มี shell/package manager → ผิวสัมผัสโจมตีน้อย
# :nonroot = รันด้วย user ที่ไม่ใช่ root (uid 65532) → ปลอดภัยขึ้น
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /bin/catice /app/catice

# backend ตัวนี้เสิร์ฟแค่ API/WebSocket — frontend อยู่ที่ Vite (dev) หรือ static host (prod)
# main.go มี FileServer("web") อยู่ แต่ถ้าไม่มี web/ ก็แค่ตอบ 404 ที่ "/" (ไม่ crash)
# ถ้าจะให้ container เสิร์ฟ frontend เองแบบ standalone:
#   1) cd ../catice-frontend && npm run build   (output → Catice2/web)
#   2) เพิ่มบรรทัด: COPY web /app/web

EXPOSE 8080
ENTRYPOINT ["/app/catice"]
</content>
