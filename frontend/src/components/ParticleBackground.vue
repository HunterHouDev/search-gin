<template>
  <canvas ref="canvasRef" class="particle-canvas"></canvas>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';

const canvasRef = ref(null);
let particles = [];
let animationId = null;

class Particle {
  constructor(canvas) {
    this.canvas = canvas;
    this.reset();
  }

  reset() {
    this.x = Math.random() * this.canvas.width;
    this.y = Math.random() * this.canvas.height;
    this.size = Math.random() * 2.5 + 0.5;
    this.baseSize = this.size;
    this.speedX = (Math.random() - 0.5) * 0.4;
    this.speedY = (Math.random() - 0.5) * 0.4;
    this.opacity = Math.random() * 0.4 + 0.15;
    this.hue = Math.random() * 60 + 230;
    this.pulsePhase = Math.random() * Math.PI * 2;
    this.pulseSpeed = Math.random() * 0.018 + 0.008;
  }

  update() {
    this.pulsePhase += this.pulseSpeed;
    const pulse = Math.sin(this.pulsePhase) * 0.5 + 0.5;
    this.size = this.baseSize + pulse * 1.2;
    this.opacity = 0.15 + pulse * 0.25;

    this.x += this.speedX;
    this.y += this.speedY;

    if (this.x < 0 || this.x > this.canvas.width) this.speedX *= -1;
    if (this.y < 0 || this.y > this.canvas.height) this.speedY *= -1;
  }

  draw(ctx) {
    ctx.save();
    ctx.globalAlpha = this.opacity;
    ctx.shadowBlur = 12;
    ctx.shadowColor = `hsl(${this.hue}, 80%, 65%)`;
    ctx.fillStyle = `hsl(${this.hue}, 70%, 72%)`;
    ctx.beginPath();
    ctx.arc(this.x, this.y, Math.max(0.1, this.size), 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();
  }
}

function initParticles() {
  if (!canvasRef.value) return;
  const canvas = canvasRef.value;
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;
  particles = [];
  const count = Math.min(350, Math.floor((canvas.width * canvas.height) / 6000));
  for (let i = 0; i < count; i++) {
    particles.push(new Particle(canvas));
  }
}

function animate() {
  if (!canvasRef.value) return;
  const canvas = canvasRef.value;
  const ctx = canvas.getContext('2d');
  ctx.fillStyle = 'rgba(8, 8, 14, 0.12)';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  particles.forEach((p) => {
    p.update();
    p.draw(ctx);
  });
  animationId = requestAnimationFrame(animate);
}

function handleResize() {
  initParticles();
}

onMounted(() => {
  initParticles();
  animate();
  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId);
  }
  window.removeEventListener('resize', handleResize);
});
</script>

<style scoped>
.particle-canvas {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
}
</style>
