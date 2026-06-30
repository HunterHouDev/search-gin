<template>
  <q-dialog ref="dialogRef" @hide="hide" @close="hide" :maximized="true">
    <q-layout view="lHh Lpr lFf" container style="height: 80vh;"
      :style="{ width: isMobile() ? '100%' : '800px', ...themeStyle }">
      <q-header>
        <q-toolbar class="bg-black text-white shadow-2 rounded-borders w100 justify-between">
          <q-btn color="red" dense flat icon="ti-shift-left" @click="prevOne">
            <q-tooltip class="bg-white text-primary">上一个</q-tooltip>
          </q-btn>
          <q-tabs ripple v-model="tab" align="justify" style="width: 60%"
            :active-color="systemProperty.theme === 'natural' ? 'primary' : 'white'"
            :indicator-color="systemProperty.theme === 'natural' ? 'green' : 'white'"
            @update:model-value="view.startTime = '00:00:05'">
            <q-tab name="png" label="png" />
            <q-tab name="jpg" label="jpg" />
            <q-tab name="cut" label="cut" />
          </q-tabs>
          <q-btn flat dense icon="ti-reload" color="red" @click="closeWin">
            <q-tooltip class="bg-white text-primary">刷新</q-tooltip>
          </q-btn>
          <q-btn color="red" dense flat icon="ti-shift-right" @click="nextOne">
            <q-tooltip class="bg-white text-primary">下一个</q-tooltip>
          </q-btn>
        </q-toolbar>
        <p style="
            display: -webkit-box; /* 将对象作为弹性伸缩盒子模型显示 */
            -webkit-box-orient: vertical; /* 设置子元素的排列方式为垂直方向 */
            line-clamp: 2; /* 设置显示的行数 */
            overflow: hidden; /* 隐藏溢出文本 */
            text-overflow: ellipsis; /* 显示省略号 */
            padding: 4px 4px;
          ">
          {{ view.item.Name }}
        </p>
        <div style="
            display: flex;
            flex-direction: row;
            justify-content: center;
            padding: 0;
          " :style="themeStyle">
          <q-input v-model="view.startTime" mask="fulltime" :rules="['fulltime']" @change="previewPicture">
            <template v-slot:append>
              <div style="width: 100%">
                <q-icon name="access_time" class="cursor-pointer">
                  <q-popup-proxy cover transition-show="scale" transition-hide="scale">
                    <q-time v-model="view.startTime" with-seconds format24h>
                      <div class="row items-center justify-end">
                        <q-btn v-close-popup label="Close" color="primary" flat />
                      </div>
                    </q-time>
                  </q-popup-proxy>
                </q-icon>
              </div>
            </template>
          </q-input>
          <q-btn size="md" flat color="orange" label="截" @click="previewPicture" />
          <q-btn color="primary" v-for="(item, index) in btns" :key="index" :label="item" flat @click="timePlus(item)"
            @contextmenu="
              (e) => {
                timePlus(-item);
                e.returnValue = false;
              }
            " />
        </div>
      </q-header>

      <q-page-container style="padding: 160px 1px 1px 1px">
        <q-page>
          <q-tab-panels v-model="tab" animated style="height: 100%; overflow: auto; margin: 0; width: 100%">
            <q-tab-panel name="jpg">
              <q-img fit="fill" class="max-image-height" v-if="!view.uImage" :src="view.item.JpgUrl" />

              <q-img fit="fill" v-if="view.uImage" class="max-image-height" :src="view.uImage" />
            </q-tab-panel>
            <q-tab-panel name="png">
              <q-img fit="fill" v-show="!view.showCanvas" class="max-image-height" v-if="!view.uPng"
                :src="view.item.JpgUrl" />

              <q-img fit="fill" v-show="!view.showCanvas" v-if="view.uPng" class="max-image-height" :src="view.uPng" />
              <q-btn color="primary" flat @click="view.showCanvas = !view.showCanvas"
                :label="view.showCanvas ? '关闭裁剪' : '去裁剪'"></q-btn>
              <canvas v-show="view.showCanvas" id="mycanvas" ref="mycanvas" width="700px" height="500px"
                style="border: 1px solid #000">
              </canvas>
              <q-btn color="primary" flat @click="scalePng" label="裁剪" v-show="view.showCanvas"></q-btn>
            </q-tab-panel>
            <q-tab-panel name="cut" class="q-gutter-y-xs">
              <q-img fit="contain" v-for="item in view.prewiewImages" :key="item.Id"
                :src="GetFileByPathUseEncode(item.Path)" style="width: 100%; height: auto" class="max-image-height">
                <template v-slot:error>
                  <!-- 图片加载失败时显示的占位图 -->
                  <div class="text-subtitle1 text-white">
                    <q-icon name="image_not_supported" size="8em"></q-icon>
                    <div>图片加载失败</div>
                    <q-btn color="rgba(0,0,0,0.5)" size="sm" dense ripple @click="deleteTemp(item.Path)"
                      icon="ti-trash">
                      <q-tooltip class="bg-white text-primary">删除</q-tooltip>
                    </q-btn>
                  </div>
                </template>
                <div style="padding: 0; position: relative; float: right">
                  <q-btn color="rgba(0,0,0,0.5)" size="sm" dense ripple @click="deleteTemp(item.Path)" icon="ti-trash">
                    <q-tooltip class="bg-white text-primary">删除</q-tooltip>
                  </q-btn>
                </div>
              </q-img>
            </q-tab-panel>
          </q-tab-panels>
          <!-- 页面滚动器 -->
          <q-page-scroller position="bottom-right" :scroll-offset="150" :offset="[18, 100]">
            <q-btn fab icon="keyboard_arrow_up" color="accent" />
          </q-page-scroller>
        </q-page>
      </q-page-container>
    </q-layout>
  </q-dialog>
</template>

<script setup>
import { reactive, ref, computed } from 'vue';

import { useDialogPluginComponent, useQuasar } from 'quasar';
import {
  CutImage,
  DeleteFileByPathUseEncode,
  QueryDirImages,
  OpenFolderByPath,
} from 'components/api/searchAPI';
import { GetFileByPathUseEncode } from 'components/utils/images';
import { isMobile } from 'src/boot/platform';
import { useSystemProperty } from 'src/stores/System';

const themeStyle = computed(() => systemProperty.themeStyle);

const systemProperty = useSystemProperty()
const $q = useQuasar();
const tab = ref('png');

const btns = [0, -2, 5, 25, 60];

const view = reactive({
  item: {},
  showCanvas: false,
  startTime: '00:00:05',
  uPng: '',
  uImage: '',
});

const { dialogRef, onDialogCancel } = useDialogPluginComponent();

const open = (item) => {
  if (dialogRef.value) {
    dialogRef.value.show();
  }
  view.uPng = '';
  view.uImage = '';
  view.startTime = '00:00:05';
  view.item = item;

  const img = new Image(); // 创建一个新的图片对象
  img.crossOrigin = 'anonymous'; // 处理跨域问题
  img.src = view.item.PngUrl; // 设置图片的源地址为base64编码的图片数据
  img.onerror = () => { console.warn('图片加载失败:', view.item.PngUrl); };
  canvasData.image = img;

  img.onload = function () {
    canvasData.image = img;
    drawCanvas();
  };
  loadImage(item);
};

const loadImage = (item) => {
  if (item) {
    QueryDirImages(item.Id, 'asc').then((res) => {
      view.prewiewImages = res.data;
    });
  }
};

const deleteTemp = async (path) => {
  await DeleteFileByPathUseEncode(path);
  loadImage(view.item);
};

const canvasData = reactive({
  isDrawing: false,
  canvas: null,
  context: null,
  chooseItem: null,
  chooseAction: null,
  image: null,
  width: 1000,
  height: 1000,
  startX: 0,
  startY: 0,
  endX: 0,
  endY: 0,
});

function drawRect(rect) {
  if (!rect) return;
  canvasData.context.setLineDash([10, 5]);
  canvasData.context.lineWidth = 3;
  canvasData.context.lineCap = 'round';
  canvasData.context.globalAlpha = 0.8;
  canvasData.context.strokeStyle = 'red';
  canvasData.context.strokeRect(
    rect.startX,
    rect.startY,
    rect.width,
    rect.height
  );
}
const clearCanvas = () => {
  canvasData.context.clearRect(0, 0, canvasData.width, canvasData.height);
};

const drawImage = () => {
  let width;
  let height;
  if (canvasData.image.width > canvasData.width) {
    width = canvasData.width;
    height =
      (canvasData.image.height * canvasData.width) / canvasData.image.width;
  } else {
    width = canvasData.image.width;
    height = canvasData.image.height;
  }
  canvasData.context.drawImage(canvasData.image, 0, 0, width, height); // 将图片绘制到画布上
};
const drawLine = (e) => {
  canvasData.context.globalAlpha = 2; // 设置透明度为0.5
  // 开始一条路径
  canvasData.context.beginPath();
  // 填充色
  canvasData.context.strokeStyle = 'red';
  // 路径宽度
  canvasData.context.lineWidth = 1;
  canvasData.context.zindex = 99;
  // 移动到 鼠标的 x位置， y位置 0（竖线的起点）
  canvasData.context.moveTo(e.offsetX, 0);
  // lineTo() 方法添加一个新点，（竖线的终点）
  canvasData.context.lineTo(e.offsetX, canvasData.height);
  // 移动到(x: 0, y：鼠标的位置)（横线的起点）
  canvasData.context.moveTo(0, e.offsetY);
  // lineTo() 方法添加一个新点，（横线的终点）
  canvasData.context.lineTo(canvasData.width, e.offsetY);
  canvasData.context.stroke();
};

function finishedPosition(e) {
  canvasData.isDrawing = false;
  const { offsetX, offsetY } = e;
  canvasData.endX = canvasData.startX + offsetX;
  canvasData.endY = canvasData.startY + offsetY;
  // 绘制矩形
  const rect = {
    startX: canvasData.startX,
    startY: canvasData.startY,
    width: offsetX,
    height: offsetY,
  };
  drawRect(rect);
  canvasData.chooseAction = null;
}

function startPosition(e) {
  const { layerX, layerY } = e;
  canvasData.startX = layerX;
  canvasData.startY = layerY;
  const isMove =
    canvasData.chooseItem &&
    layerX >= canvasData.chooseItem.startX &&
    layerX <= canvasData.chooseItem.startX + canvasData.chooseItem.width &&
    layerY >= canvasData.chooseItem.startY &&
    layerY <= canvasData.chooseItem.startY + canvasData.chooseItem.height;
  if (isMove) {
    canvasData.chooseAction = 'move';
  } else {
  }
  canvasData.isDrawing = true;
}

const mouseMove = (e) => {
  clearCanvas();
  drawImage();
  if (canvasData.isDrawing) {
    const { layerX, layerY } = e;
    const offsetX = layerX - canvasData.startX;
    const offsetY = layerY - canvasData.startY;
    // console.log('offset', { offsetX, offsetY });
    if (canvasData.chooseAction == 'move') {
      canvasData.chooseItem.startX = canvasData.chooseItem.startX + offsetX / 2;
      canvasData.chooseItem.startY = canvasData.chooseItem.startY + offsetY / 2;
    } else {
      canvasData.endX = canvasData.startX + offsetX;
      canvasData.endY = canvasData.startY + offsetY;
      // 绘制矩形
      const rect = {
        startX: canvasData.startX,
        startY: canvasData.startY,
        width: offsetX,
        height: offsetY,
      };
      canvasData.chooseItem = rect;
    }
    drawRect(canvasData.chooseItem);
  } else {
    // 框、线
    drawRect(canvasData.chooseItem);
  }
  if (e) {
    drawLine(e);
  }
};

const drawCanvas = () => {
  const mycanvas = document.getElementById('mycanvas');
  if (!mycanvas) return;
  // 先移除旧监听器，避免重复绑定
  mycanvas.removeEventListener('mousedown', startPosition);
  mycanvas.removeEventListener('mouseup', finishedPosition);
  mycanvas.removeEventListener('mousemove', mouseMove);
  const ctx = mycanvas.getContext('2d');
  canvasData.canvas = mycanvas;
  canvasData.context = ctx;
  canvasData.width = mycanvas.offsetWidth;
  canvasData.height = mycanvas.offsetHeight;
  mycanvas.addEventListener('mousedown', startPosition);
  mycanvas.addEventListener('mouseup', finishedPosition);
  mycanvas.addEventListener('mousemove', mouseMove);
  drawImage();
};

const scalePng = () => {
  const mycanvas = document.getElementById('mycanvas'); // 获取canvas元素的引用
  if (!mycanvas) return;
  mycanvas.removeEventListener('mousemove', mouseMove);
  mycanvas.removeEventListener('mouseup', finishedPosition);
  mycanvas.removeEventListener('mousedown', startPosition);
  if (canvasData.chooseItem) {
    const { startX, startY, width, height } = canvasData.chooseItem;
    var croppedCanvas = document.createElement('canvas'); // 创建新的Canvas用于剪切后的图片
    var croppedCtx = croppedCanvas.getContext('2d');
    croppedCtx.drawImage(
      canvasData.canvas,
      startX,
      startY,
      width,
      height,
      0,
      0,
      width,
      height
    );
    // 将剪切后的图片保存为JPEG格式的文件
    const pngPath = view.item.Title + '.png';
    croppedCanvas.toBlob(async function (blob) {
      var a = document.createElement('a');
      a.download = pngPath; // 设置下载的文件名
      a.href = URL.createObjectURL(blob); // 创建下载链接
      document.body.appendChild(a); // 将链接添加到文档中
      a.click(); // 模拟点击下载链接
      document.body.removeChild(a); // 移除链接
      URL.revokeObjectURL(a.href); // 释放URL对象
    });
    OpenFolderByPath(view.item.DirPath);
    // OpenFolerByPath({ dirpath: 'downloads' });
  }
};

const previewPicture = async () => {
  if (view.startTime) {
    if (tab.value == 'png') {
      const { Data } = await CutImage(view.item.Id, 'Png', view.startTime, false);
      view.uPng = `data:image/png;base64,${Data}`;
      const img = new Image(); // 创建一个新的图片对象
      img.crossOrigin = 'anonymous'; // 处理跨域问题
      img.src = `data:image/png;base64,${Data}`;
      img.onload = function () {
        canvasData.image = img;
        clearCanvas();
        drawImage();
      };
    } else if (tab.value == 'jpg') {
      const { Data } = await CutImage(view.item.Id, 'Jpg', view.startTime, false);
      view.uImage = `data:image/jpeg;base64,${Data}`;
    } else if (tab.value == 'cut') {
      await CutImage(view.item.Id, 'shot', view.startTime, false);
      loadImage(view.item);
    }
    $q.notify({ message: '已执行', position: 'bottom-left' });

  }
};

const timePlus = (n) => {
  if (n !== 0) {
    view.startTime = plusN(view.startTime, n);
  } else {
    view.startTime = '00:00:00';
  }

  previewPicture();
};

const plusN = (base, n) => {
  const baseArr = base.split(':');
  let baseNum = 0;
  if (baseArr.length == 1) {
    baseNum = parseInt(baseArr[0]);
  } else if (baseArr.length == 2) {
    baseNum = parseInt(baseArr[0]) * 60 + parseInt(baseArr[1]);
  } else if (baseArr.length == 3) {
    baseNum =
      parseInt(baseArr[0]) * 3600 +
      parseInt(baseArr[1]) * 60 +
      parseInt(baseArr[2]);
  }
  baseNum += n;
  const hh = (parseInt(String(baseNum / 3600)) + ':').padStart(3, '0');
  const mm = (parseInt(String((baseNum % 3600) / 60)) + ':').padStart(3, '0');
  const ss = ((baseNum % 60) + '').padStart(2, '0');
  return hh + mm + ss;
};

const emits = defineEmits(['next-one', 'prev-one', 'hide']);

const nextOne = async () => {
  emits('next-one');
};

const prevOne = async () => {
  emits('prev-one');
};

const closeWin = () => {
  emits('hide');
  onDialogCancel();
  window.location.reload();
};

const hide = () => {
  emits('hide');
  onDialogCancel();
};

defineExpose({
  open,
});
</script>

<style scoped>
.max-image-height {
  max-height: 640px;
}
</style>
