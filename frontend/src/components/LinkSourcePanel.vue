<template>
  <div
    class="link-source-panel"
    :class="{
      'link-source-panel-split': embedded,
      // 播放中：左列（分片列表 + 下载列表）与右列（播放器）并排
      'link-source-panel-playing': embedded && embeddedActive && !playerStopped,
    }"
  >
    <!-- 链接类型 Tab -->
    <div class="link-tabs">
      <button
        v-for="tab in visibleTabs"
        :key="tab.value"
        type="button"
        class="link-tab"
        :class="{ 'link-tab-active': linkTab === tab.value }"
        @click="switchLinkTab(tab.value)"
      >
        <q-icon :name="tab.icon" size="15px" />
        <span>{{ tab.label }}</span>
      </button>
    </div>

    <!-- 链接输入框 -->
    <div
      class="magnet-input-wrapper"
      :class="{
        'magnet-focused': linkFocused,
        'magnet-input-multi': linkTab === 'hls',
      }"
    >
      <q-icon
        :name="activeLinkTab.icon"
        color="indigo-4"
        size="20px"
        class="magnet-icon"
      />
      <!-- 分片链接：多行输入，每行一个地址（带排序号，输入后自动生成下一行） -->
      <div v-if="linkTab === 'hls'" class="hls-url-rows">
        <div
          v-for="(row, index) in hlsUrlRows"
          :key="row.id"
          class="hls-url-row"
        >
          <span class="hls-url-order">{{ index + 1 }}</span>
          <q-input
            :model-value="row.value"
            :placeholder="
              index === 0
                ? activeLinkTab.placeholder
                : '继续输入或粘贴 m3u8 地址（可一次粘贴多个）'
            "
            dark
            dense
            borderless
            class="hls-url-field"
            :disable="hlsParsing"
            @update:model-value="
              (val) => updateHlsUrlRow(row.id, String(val ?? ''))
            "
            @keyup.enter="submitLink"
            @focus="linkFocused = true"
            @blur="linkFocused = false"
          />
        </div>
      </div>
      <q-input
        v-else
        v-model="activeLinkValue"
        :placeholder="activeLinkTab.placeholder"
        dark
        dense
        borderless
        class="magnet-input"
        @keyup.enter="submitLink"
        @focus="linkFocused = true"
        @blur="linkFocused = false"
      />
      <q-btn
        v-if="linkTab === 'hls' && canSubmitLink"
        flat
        dense
        round
        size="sm"
        color="indigo-4"
        icon="backspace"
        :disable="hlsParsing"
        @click="clearHlsUrlRows"
        class="hls-url-clear-btn"
      >
        <q-tooltip class="bg-dark text-white">清空全部地址</q-tooltip>
      </q-btn>
      <q-btn
        flat
        dense
        no-caps
        color="indigo-4"
        :icon="linkActionIcon"
        :label="linkActionLabel"
        size="sm"
        @click="submitLink"
        :disable="!canSubmitLink"
        :loading="linkActionLoading"
        class="magnet-submit-btn"
      >
        <q-tooltip class="bg-dark text-white">{{
          linkActionTooltip
        }}</q-tooltip>
      </q-btn>
    </div>

    <!-- 分片列表：解析后可删除分片（如广告）再播放剩余部分 -->
    <transition name="fade">
      <div
        v-if="linkTab === 'hls' && (hlsParsed || hlsDownloadList.length)"
        class="hls-segment-panel"
      >
        <template v-if="hlsParsed">
          <div class="hls-segment-header">
            <q-icon name="playlist_play" size="16px" color="indigo-4" />
            <span class="hls-segment-summary">
              <template v-if="hlsSourceCount > 1"
                >{{ hlsSourceCount }} 个源 · </template
              >保留 {{ hlsKeptCount }}/{{ hlsTotalCount }} 个分片 ·
              {{ hlsKeptDuration }}
              <template v-if="hlsRemovedCount"
                >（已删除 {{ hlsRemovedCount }}）</template
              >
              <template v-if="hlsAdCount"
                >（广告 {{ hlsAdCount }}，已排除）</template
              >
            </span>
            <q-space />
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="indigo-4"
              icon="restart_alt"
              label="恢复"
              :disable="!hlsRemovedCount"
              @click="restoreHlsSegments"
            >
              <q-tooltip class="bg-dark text-white"
                >恢复全部已删除分片（广告仍保持置灰并排除）</q-tooltip
              >
            </q-btn>
            <!-- 广告黑名单：可查看 / 取消单条 / 清空 -->
            <q-btn
              v-if="hlsAdBlacklist.length"
              flat
              dense
              no-caps
              size="sm"
              color="negative"
              icon="block"
              :label="`广告黑名单 ${hlsAdBlacklist.length}`"
            >
              <q-tooltip class="bg-dark text-white"
                >被标记为广告的分片类：列表中置灰，下载 / 播放时忽略</q-tooltip
              >
              <q-menu dark class="hls-ad-menu">
                <q-list dense>
                  <q-item
                    v-for="key in hlsAdBlacklist"
                    :key="key"
                    clickable
                    v-close-popup
                    @click="unmarkHlsAdSegments(key)"
                  >
                    <q-item-section>
                      <q-item-label class="hls-ad-menu-label">{{
                        key
                      }}</q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-icon name="undo" size="16px" />
                    </q-item-section>
                  </q-item>
                  <q-separator dark />
                  <q-item clickable v-close-popup @click="clearHlsAdBlacklist">
                    <q-item-section>
                      <q-item-label>清空黑名单</q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-icon name="delete_forever" size="16px" />
                    </q-item-section>
                  </q-item>
                </q-list>
              </q-menu>
            </q-btn>
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="indigo-4"
              icon="link"
              label="复制链接"
              :disable="!hlsKeptCount"
              @click="copyAllSegmentUrls"
            >
              <q-tooltip class="bg-dark text-white"
                >复制保留的 {{ hlsKeptCount }} 个分片链接（每行一个）</q-tooltip
              >
            </q-btn>
            <q-btn
              unelevated
              dense
              no-caps
              size="sm"
              color="indigo-6"
              icon="play_arrow"
              label="播放"
              class="hls-play-btn"
              :loading="hlsLoading"
              :disable="!hlsKeptCount"
              @click="playHlsRemaining"
            >
              <q-tooltip class="bg-dark text-white">{{
                hlsSourceCount > 1
                  ? `按顺序合并播放 ${hlsSourceCount} 个源的 ${hlsKeptCount} 个分片`
                  : `播放剩余 ${hlsKeptCount} 个分片`
              }}</q-tooltip>
            </q-btn>
          </div>

          <!-- 源列表：多个 m3u8 按这里显示的顺序合并，可上移 / 下移 / 移除 -->
          <div v-if="hlsSourceList.length > 1" class="hls-source-list">
            <div
              v-for="(item, index) in hlsSourceList"
              :key="item.source.id"
              class="hls-source-item"
            >
              <span class="hls-source-order">{{ index + 1 }}</span>
              <span class="hls-source-url" :title="item.source.url">{{
                item.source.url
              }}</span>
              <span class="hls-source-meta"
                >{{ item.keptCount }}/{{ item.totalCount }} 个分片 ·
                {{ item.duration }}</span
              >
              <q-btn
                flat
                round
                dense
                size="sm"
                color="indigo-4"
                icon="arrow_upward"
                :disable="index === 0"
                @click="moveHlsSource(item.source.id, -1)"
              >
                <q-tooltip class="bg-dark text-white">上移</q-tooltip>
              </q-btn>
              <q-btn
                flat
                round
                dense
                size="sm"
                color="indigo-4"
                icon="arrow_downward"
                :disable="index === hlsSourceList.length - 1"
                @click="moveHlsSource(item.source.id, 1)"
              >
                <q-tooltip class="bg-dark text-white">下移</q-tooltip>
              </q-btn>
              <q-btn
                flat
                round
                dense
                size="sm"
                color="red-4"
                icon="delete_outline"
                @click="removeHlsSource(item.source.id)"
              >
                <q-tooltip class="bg-dark text-white"
                  >移除该源及其分片</q-tooltip
                >
              </q-btn>
            </div>
          </div>
          <!-- 下载设置：文件名 + 目录 + 下载按钮，撑满一行（文件名自适应剩余宽度） -->
          <div class="hls-download-row">
            <q-input
              v-model="hlsDownloadName"
              dark
              dense
              borderless
              class="hls-filename-input"
              :placeholder="hlsDefaultDownloadName"
              :disable="!hlsKeptCount"
              maxlength="120"
              @keyup.enter="downloadHls"
            >
              <template #prepend>
                <q-icon name="edit_note" size="16px" color="indigo-4" />
              </template>
              <q-tooltip class="bg-dark text-white">
                自定义保存文件名，留空则用默认名
                {{ hlsDefaultDownloadName }}
              </q-tooltip>
            </q-input>
            <q-btn-dropdown
              flat
              dense
              no-caps
              size="sm"
              class="hls-dir-btn"
              :color="hlsDownloadDir ? 'green-4' : 'indigo-4'"
              :icon="hlsDownloadDir ? 'folder_special' : 'create_new_folder'"
              :label="hlsDownloadDir || '下载目录'"
            >
              <q-list dense class="hls-dir-menu">
                <q-item
                  clickable
                  v-close-popup
                  @click="chooseHlsDownloadDir('')"
                >
                  <q-item-section>
                    <q-item-label>默认目录</q-item-label>
                    <q-item-label caption
                      >服务端默认保存位置（第一个媒体目录）</q-item-label
                    >
                  </q-item-section>
                </q-item>
                <q-item
                  v-for="dir in hlsDownloadDirOptions"
                  :key="dir"
                  clickable
                  v-close-popup
                  @click="chooseHlsDownloadDir(dir)"
                >
                  <q-item-section>
                    <q-item-label class="hls-dir-menu-label">{{
                      dir
                    }}</q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
              <q-tooltip class="bg-dark text-white">
                {{
                  hlsDownloadDir
                    ? `下载由服务端存入「${hlsDownloadDir}」；点击可更换`
                    : '选择服务端保存目录，下载由服务端执行'
                }}
              </q-tooltip>
            </q-btn-dropdown>
            <!-- 下载完成后自动转码：默认不转，选中后由服务端在落盘后按路径建转码任务 -->
            <q-btn-dropdown
              flat
              dense
              no-caps
              size="sm"
              class="hls-xcode-btn"
              :color="
                hlsXcodeSkipped
                  ? 'grey-6'
                  : hlsDownloadXcode
                    ? 'teal-4'
                    : 'indigo-4'
              "
              icon="transform"
              :label="hlsDownloadXcodeLabel"
              :disable="!hlsKeptCount || hlsXcodeSkipped"
            >
              <q-list dense class="hls-xcode-menu">
                <q-item
                  v-for="opt in hlsXcodeOptions"
                  :key="opt.value"
                  clickable
                  v-close-popup
                  @click="hlsDownloadXcode = opt.value"
                >
                  <q-item-section>
                    <q-item-label>{{ opt.label }}</q-item-label>
                    <q-item-label caption>{{ opt.caption }}</q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
              <q-tooltip class="bg-dark text-white">{{
                hlsXcodeSkipped
                  ? '原视频已是 MP4（fMP4 源），下载产物即为 mp4，无需转码'
                  : '下载完成后由服务端自动转码（转码是独立任务，可在任务列表查看进度）'
              }}</q-tooltip>
            </q-btn-dropdown>
            <!-- 并发下载数量：单个任务内同时下载的分片数，随任务一起提交给服务端 -->
            <q-input
              v-model="concurrencyModel"
              dark
              dense
              borderless
              type="number"
              :min="DOWNLOAD_PARAM_MIN"
              :max="DOWNLOAD_PARAM_MAX"
              class="hls-param-input"
              prefix="并发"
              :disable="!hlsKeptCount"
            >
              <template #prepend>
                <q-icon name="speed" size="16px" color="indigo-4" />
              </template>
              <q-tooltip class="bg-dark text-white"
                >并发下载数量：该任务内同时下载的分片数（1~16，默认 4）。源站频繁断连
                / 限流时调小更稳，网络好时调大更快</q-tooltip
              >
            </q-input>
            <!-- 并行任务数量：服务端同时执行的任务数上限，对所有任务类型生效 -->
            <q-input
              v-model="parallelModel"
              dark
              dense
              borderless
              type="number"
              :min="DOWNLOAD_PARAM_MIN"
              :max="DOWNLOAD_PARAM_MAX"
              class="hls-param-input"
              prefix="并行"
              :disable="!hlsKeptCount"
            >
              <template #prepend>
                <q-icon name="tune" size="16px" color="indigo-4" />
              </template>
              <q-tooltip class="bg-dark text-white"
                >并行任务数量：服务端同时执行的任务数上限（1~16，默认
                4），对所有任务类型生效，提交下载时一并应用</q-tooltip
              >
            </q-input>
            <!-- 点下载后任务交给服务端执行，本地分片列表随之清空，可继续粘贴下一组地址 -->
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="indigo-4"
              icon="download"
              label="下载"
              class="hls-download-btn"
              :disable="!hlsKeptCount"
              @click="downloadHls"
            >
              <q-tooltip class="bg-dark text-white"
                >由服务端下载 {{ hlsKeptCount }} 个分片并合并保存，关闭窗口 /
                刷新页面都不会中断</q-tooltip
              >
            </q-btn>
            <!-- 浏览器直下（前端备用）：放在服务端下载按钮旁边，不经服务端直接拉当前分片列表合并另存 -->
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="deep-orange-4"
              icon="cloud_download"
              :label="browserNowLabel"
              class="hls-download-btn"
              :disable="!hlsKeptCount || browserNowInProgress()"
              @click="downloadHlsInBrowserNow"
            >
              <q-tooltip class="bg-dark text-white"
                >浏览器直接下载当前 {{ hlsKeptCount }} 个分片并合并另存，不经服务端；源站禁止跨域时失败，此为本页关闭后下载即中断</q-tooltip
              >
            </q-btn>
          </div>
          <div class="hls-segment-list">
            <template
              v-for="(group, groupIndex) in hlsSegmentGroups"
              :key="group.sourceId"
            >
              <!-- 多源时标出每段的边界，便于分辨删除的是哪一个源的分片 -->
              <div v-if="hlsSourceCount > 1" class="hls-segment-group">
                <span class="hls-segment-group-label"
                  >源 {{ groupIndex + 1 }}</span
                >
                <span class="hls-segment-group-url" :title="group.url">{{
                  group.url
                }}</span>
                <span class="hls-segment-group-count"
                  >{{ group.segments.length }} 个分片</span
                >
              </div>
              <!-- 黑名单（广告）分片仍在列表中，但置灰 + 未勾选，下载 / 播放时忽略 -->
              <div
                v-for="seg in group.segments"
                :key="seg.id"
                class="hls-segment-item"
                :class="{ 'hls-segment-item--ad': isHlsAdSegment(seg) }"
              >
                <q-icon
                  class="hls-segment-check"
                  :name="
                    isHlsAdSegment(seg)
                      ? 'check_box_outline_blank'
                      : 'check_box'
                  "
                  size="16px"
                  :color="isHlsAdSegment(seg) ? 'grey' : 'indigo-4'"
                >
                  <q-tooltip class="bg-dark text-white">{{
                    isHlsAdSegment(seg)
                      ? '广告（黑名单）：不参与播放 / 下载'
                      : '参与播放 / 下载'
                  }}</q-tooltip>
                </q-icon>
                <span class="hls-segment-index">#{{ seg.index }}</span>
                <span class="hls-segment-duration">{{
                  seg.duration ? seg.duration.toFixed(1) + 's' : '--'
                }}</span>
                <span
                  class="hls-segment-url"
                  :title="seg.url"
                  @click="copySegmentUrl(seg)"
                  >{{ seg.url }}</span
                >
                <q-btn
                  flat
                  round
                  dense
                  size="sm"
                  color="indigo-4"
                  icon="content_copy"
                  @click="copySegmentUrl(seg)"
                >
                  <q-tooltip class="bg-dark text-white"
                    >复制该分片链接</q-tooltip
                  >
                </q-btn>
                <!-- 标记 / 取消广告：同类分片进黑名单，列表置灰且下载时忽略 -->
                <q-btn
                  flat
                  round
                  dense
                  size="sm"
                  :color="isHlsAdSegment(seg) ? 'grey' : 'negative'"
                  :icon="isHlsAdSegment(seg) ? 'undo' : 'block'"
                  @click="toggleHlsAdSegment(seg.id)"
                >
                  <q-tooltip class="bg-dark text-white">{{
                    isHlsAdSegment(seg)
                      ? '取消广告标记：该类分片恢复参与播放 / 下载'
                      : '标记为广告：同类分片加入黑名单（列表置灰，下载时忽略）'
                  }}</q-tooltip>
                </q-btn>
                <q-btn
                  flat
                  round
                  dense
                  size="sm"
                  color="deep-orange-4"
                  icon="delete_sweep"
                  @click="removeHlsSimilarSegments(seg.id)"
                >
                  <q-tooltip class="bg-dark text-white">
                    删除同类分片（最后一节不同、前面都相同的全部删除）
                  </q-tooltip>
                </q-btn>
                <q-btn
                  flat
                  round
                  dense
                  size="sm"
                  color="red-4"
                  icon="delete_outline"
                  @click="removeHlsSegment(seg.id)"
                >
                  <q-tooltip class="bg-dark text-white">删除该分片</q-tooltip>
                </q-btn>
              </div>
            </template>
          </div>
        </template>

      </div>
    </transition>

    <!-- 播放区 / 下载区：左右布局；左列下载列表，播放区在点「播放」后于右列展开 -->
    <div class="link-source-stage" :class="{ 'link-source-stage-split': embedded }">
      <!-- 左列：下载列表（点「下载」即出现在这里；重新添加链接不会清空） -->
      <div
        v-if="linkTab === 'hls' && hlsDownloadList.length"
        class="link-source-stage-download"
      >
        <div class="hls-download-list-panel">
          <div class="hls-download-list-header">
            <q-icon name="download_done" size="16px" color="green-4" />
            <span
              class="hls-download-list-title hls-download-stat-btn"
              :class="{ 'hls-download-stat-active': hlsDownloadFilter === 'all' }"
              @click="toggleDownloadFilter('all')"
              >下载列表 · {{ hlsDownloadStats.total }}</span
            >
            <!-- 点状态即筛选该状态的条目，再点一次取消筛选 -->
            <span
              class="hls-download-stat hls-download-stat-running hls-download-stat-btn"
              :class="{
                'hls-download-stat-active': hlsDownloadFilter === 'downloading',
              }"
              @click="toggleDownloadFilter('downloading')"
            >
              执行中 {{ hlsDownloadStats.downloading }}
            </span>
            <span
              class="hls-download-stat hls-download-stat-done hls-download-stat-btn"
              :class="{ 'hls-download-stat-active': hlsDownloadFilter === 'done' }"
              @click="toggleDownloadFilter('done')"
            >
              完成 {{ hlsDownloadStats.done }}
            </span>
            <span
              v-if="hlsDownloadStats.failed || hlsDownloadFilter === 'failed'"
              class="hls-download-stat hls-download-stat-failed hls-download-stat-btn"
              :class="{ 'hls-download-stat-active': hlsDownloadFilter === 'failed' }"
              @click="toggleDownloadFilter('failed')"
            >
              失败 {{ hlsDownloadStats.failed }}
            </span>
            <!-- 队列中：并行任务数已满，等有空闲槽位才真正开始 -->
            <span
              v-if="hlsDownloadStats.queued || hlsDownloadFilter === 'downloading'"
              class="hls-download-stat hls-download-stat-queued hls-download-stat-btn"
              :class="{
                'hls-download-stat-active': hlsDownloadFilter === 'downloading',
              }"
              @click="toggleDownloadFilter('downloading')"
            >
              队列中 {{ hlsDownloadStats.queued }}
            </span>
            <!-- 下载参数：默认只读；点「修改参数」切换输入态，保存后回到只读 -->
            <span v-if="!paramsEditing" class="hls-download-params">
              并发 {{ hlsDownloadConcurrency }} · 并行 {{ hlsDownloadParallel }}
              <q-btn
                flat
                dense
                no-caps
                size="sm"
                color="indigo-4"
                icon="edit"
                label="修改参数"
                @click="startEditParams"
              >
                <q-tooltip class="bg-dark text-white"
                  >并发：单个任务内同时下载的分片数（1~16）；并行：服务端同时执行的任务数上限（1~16）</q-tooltip
                >
              </q-btn>
            </span>
            <span v-else class="hls-download-params">
              <q-input
                v-model="paramDraft.concurrency"
                dark
                dense
                borderless
                type="number"
                :min="DOWNLOAD_PARAM_MIN"
                :max="DOWNLOAD_PARAM_MAX"
                prefix="并发"
                class="hls-param-input"
              />
              <q-input
                v-model="paramDraft.parallel"
                dark
                dense
                borderless
                type="number"
                :min="DOWNLOAD_PARAM_MIN"
                :max="DOWNLOAD_PARAM_MAX"
                prefix="并行"
                class="hls-param-input"
              />
              <q-btn
                flat
                dense
                no-caps
                size="sm"
                color="green-4"
                icon="check"
                label="保存"
                @click="saveParams"
              />
              <q-btn
                flat
                dense
                no-caps
                size="sm"
                color="grey-5"
                label="取消"
                @click="paramsEditing = false"
              />
            </span>
            <q-space />
            <!-- 清空已完成 / 清空已失败：只删对应状态的任务记录，不删已下载的文件 -->
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="green-4"
              icon="delete_sweep"
              label="清空已完成"
              :disable="!hlsDownloadStats.done"
              @click="clearDoneHlsDownloads"
            >
              <q-tooltip class="bg-dark text-white"
                >删除已完成的任务记录，不会删除已下载的文件</q-tooltip
              >
            </q-btn>
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="red-4"
              icon="delete_sweep"
              label="清空已失败"
              :disable="!hlsDownloadStats.failed"
              @click="clearFailedHlsDownloads"
            >
              <q-tooltip class="bg-dark text-white"
                >删除失败与已取消的任务记录，不会删除已下载的文件</q-tooltip
              >
            </q-btn>
          </div>
          <div class="hls-download-list">
            <div
              v-for="item in hlsVisibleDownloadList"
              :key="item.id"
              class="hls-download-item"
              :class="[
                `hls-download-item-${hlsDownloadBucket(item)}`,
                {
                  'hls-download-item-playing': item.id === hlsPlayingDownloadId,
                },
              ]"
            >
              <q-icon
                name="movie"
                size="16px"
                :color="hlsDownloadIconColor(item)"
                class="hls-download-item-icon"
              />
              <div class="hls-download-item-info">
                <span class="hls-download-item-name" :title="item.target">{{
                  item.name
                }}</span>
                <span class="hls-download-item-meta">{{
                  hlsDownloadMeta(item)
                }}</span>
                <q-linear-progress
                  v-if="item.status === 'downloading' || browserDownloadInProgress(item)"
                  :value="
                    item.status === 'downloading'
                      ? item.progress / 100
                      : (browserProgressRatio(item) ?? 0)
                  "
                  size="3px"
                  :color="hlsDownloadIconColor(item)"
                  track-color="rgba(148, 163, 184, 0.2)"
                  class="hls-download-item-bar"
                />
              </div>
              <!-- 取消：下载中中止任务，已结束则移除这条记录 -->
              <q-btn
                flat
                round
                dense
                size="sm"
                :color="item.status === 'downloading' ? 'red-4' : 'grey-5'"
                :icon="
                  item.status === 'downloading' ? 'close' : 'delete_outline'
                "
                @click="cancelHlsDownload(item.id)"
              >
                <q-tooltip class="bg-dark text-white">
                  {{
                    item.status === 'downloading'
                      ? '取消服务端下载任务'
                      : '移除该任务（不删除已下载的文件）'
                  }}
                </q-tooltip>
              </q-btn>
              <!-- 重启：失败/已取消的任务复用服务端保存的播放列表重新下载 -->
              <q-btn
                v-if="item.status === 'failed' || item.status === 'canceled'"
                flat
                round
                dense
                size="sm"
                color="orange-4"
                icon="restart_alt"
                @click="restartHlsDownload(item.id)"
              >
                <q-tooltip class="bg-dark text-white"
                  >重新启动该下载任务</q-tooltip
                >
              </q-btn>
              <!-- 浏览器下载（前端 JS）：优先用服务端保存的播放列表副本，
                   拿不到（失败任务早期中断）就用当前面板已解析的分片；
                   全程在浏览器内拉取分片、解密合并后另存本机，不经服务端任务 -->
              <q-btn
                flat
                round
                dense
                size="sm"
                color="orange-4"
                icon="download_for_offline"
                :disable="browserDownloadInProgress(item)"
                @click="downloadHlsInBrowser(item)"
              >
                <q-tooltip class="bg-dark text-white">
                  浏览器下载（前端直接拉取分片并存到本机，不经服务端；源站禁止跨域时失败）
                </q-tooltip>
              </q-btn>
              <!-- 在线播放：直接播源站地址，不依赖下载是否完成 -->
              <q-btn
                flat
                round
                dense
                size="sm"
                color="teal-4"
                icon="ondemand_video"
                :disable="!item.sourceUrl"
                @click="playHlsDownloadOnline(item)"
              >
                <q-tooltip class="bg-dark text-white">
                  {{
                    item.sourceUrl
                      ? '在线播放源站链接（m3u8 走 HLS；源站需可访问，签名过期会播不了）'
                      : '该任务没有源地址，无法在线播放'
                  }}
                </q-tooltip>
              </q-btn>
              <!-- 下载后播放：回放服务端已下载的文件 -->
              <q-btn
                flat
                round
                dense
                size="sm"
                color="green-4"
                icon="play_arrow"
                :disable="item.status !== 'done' || !item.playable"
                @click="playHlsDownload(item)"
              >
                <q-tooltip class="bg-dark text-white">
                  {{
                    item.status !== 'done'
                      ? '服务端下载完成后可播放'
                      : item.playable
                        ? '播放已下载的视频'
                        : isPlayableDownloadPath(item.path)
                          ? '该文件不在媒体目录内，无法在页面内回放'
                          : 'TS 容器浏览器无法直接播放：下载时选择「转 MP4」，完成后即可回放'
                  }}
                </q-tooltip>
              </q-btn>
            </div>
          </div>
        </div>
      </div>
    </div>
    <!-- 右列：内嵌播放器（宿主不提供 video 元素时由组件自己承接播放）。
         只显示/隐藏而不销毁：video 元素常驻，任何时刻都能拿到它播放。
         点「播放」才展开，解析完成不显示；点「停止」后再次隐藏。
         与左列（分片列表 + 下载列表）并排，两者共同占满面板宽度 -->
    <div
      v-if="embedded"
      v-show="embeddedActive && !playerStopped"
      class="link-source-stage-play"
    >
      <div class="link-source-player">
        <video
          ref="internalVideoRef"
          class="link-source-video"
          controls
          playsinline
          preload="metadata"
        ></video>
        <!-- 停止：断开播放并隐藏播放器（HLS 会停止继续拉分片） -->
        <q-btn
          flat
          round
          dense
          size="sm"
          color="red-4"
          icon="stop_circle"
          class="link-source-player-stop"
          @click="stopAndHidePlayer"
        >
          <q-tooltip class="bg-dark text-white"
            >停止播放并隐藏播放器</q-tooltip
          >
        </q-btn>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { useQuasar } from 'quasar';
import { useClipboard } from '@vueuse/core';
import {
  LINK_TABS,
  useLinkPlayback,
  isPlayableDownloadPath,
  type HlsDownloadItem,
  type HlsSegment,
  type LinkTab,
} from 'src/composables/useLinkPlayback';
import { useSystemProperty } from 'src/stores/System';

// 外部链接面板（视频直链 / HLS 分片链，磁力链可选）
// 业务逻辑全部来自 useLinkPlayback，本组件只负责 UI 与「播放落到哪里」：
//   1. 宿主提供 video 元素（沉浸式播放页）→ onPlay / getVideoEl / getVolume 由宿主注入；
//   2. 宿主不提供（批量编辑弹窗）→ embedded 打开自带播放器，自己接管播放。

interface Props {
  /** 需要展示的链接类型 */
  tabs?: LinkTab[];
  /** 是否渲染并使用组件自带的内嵌播放器 */
  embedded?: boolean;
  /** 磁力链输入值（v-model:magnet-uri） */
  magnetUri?: string;
  /** 提交磁力链（解析种子） */
  submitMagnet?: () => void | Promise<void>;
  /** 宿主提供播放元素 */
  getVideoEl?: () => HTMLVideoElement | null;
  /** 宿主当前音量（0~1） */
  getVolume?: () => number;
  /** 宿主自行处理播放请求 */
  onPlay?: (src: string, name: string, isHls: boolean) => void;
}

const props = withDefaults(defineProps<Props>(), {
  // 默认只展示视频链接与分片链接（磁力链需宿主提供 magnet-uri / submit-magnet）
  tabs: () => ['video', 'hls'] as LinkTab[],
  embedded: false,
  magnetUri: '',
});

const emit = defineEmits<{
  (e: 'update:magnetUri', value: string): void;
}>();

const $q = useQuasar();
const systemProperty = useSystemProperty();
// 剪贴板：legacy 兜底——非安全上下文（http 局域网访问）下 navigator.clipboard 不存在
const { copy: copyText } = useClipboard({ legacy: true });

// ── 内嵌播放器 ────────────────────────────────────────────────────────────────
const internalVideoRef = ref<HTMLVideoElement | null>(null);
/** 内嵌播放器是否已有播放内容 */
const embeddedActive = ref(false);
/** 用户点「停止」后隐藏播放器（再次点播放时恢复） */
const playerStopped = ref(false);

function getVideoEl(): HTMLVideoElement | null {
  return props.getVideoEl?.() ?? internalVideoRef.value;
}

/**
 * 兜底：播放器若尚未显示（例如外部把 embeddedActive 置回 false），
 * 先把播放器展开再返回 video 元素。播放器用 v-show 常驻，正常播放不会走到这里。
 */
async function ensureVideo(): Promise<HTMLVideoElement | null> {
  if (props.getVideoEl) return props.getVideoEl();
  if (!props.embedded) return internalVideoRef.value;
  if (embeddedActive.value && !playerStopped.value) {
    return internalVideoRef.value;
  }
  playerStopped.value = false;
  embeddedActive.value = true;
  await nextTick();
  return internalVideoRef.value;
}

function getVolume(): number {
  return props.getVolume?.() ?? internalVideoRef.value?.volume ?? 0.8;
}

/** 播放请求：宿主有 onPlay 交给宿主，否则写入内嵌播放器 */
async function handlePlay(src: string, name: string, isHls: boolean) {
  if (props.onPlay) {
    await props.onPlay(src, name, isHls);
    return;
  }
  // 播放器按需显示：先置为激活，等这一帧渲染生效再写 src
  playerStopped.value = false;
  embeddedActive.value = isHls || src.length > 0;
  await nextTick();
  const videoEl = internalVideoRef.value;
  if (!videoEl) return;
  videoEl.title = name;
  // src 为空表示由 composable 内部的 hls.js 实例接管：
  // 这里不能 load()——它会清掉随后 attachMedia 写入的媒体源
  if (!src) return;
  videoEl.pause();
  videoEl.removeAttribute('src');
  videoEl.load();
  videoEl.src = src;
  videoEl.play().catch((e: Error) => {
    console.warn('Autoplay blocked:', e.message);
  });
}

/** 磁力链输入框双向绑定（宿主未开放磁力链 tab 时不会被使用） */
const magnetModel = computed<string>({
  get: () => props.magnetUri,
  set: (value) => emit('update:magnetUri', value),
});

const {
  linkTab,
  linkFocused,
  activeLinkTab,
  activeLinkValue,
  canSubmitLink,
  hlsUrlRows,
  updateHlsUrlRow,
  clearHlsUrlRows,
  hlsLoading,
  // 解析中：模板用它禁用解析按钮，缺了会一直是 undefined
  hlsParsing,
  linkActionLabel,
  linkActionIcon,
  linkActionTooltip,
  linkActionLoading,
  hlsParsed,
  hlsSegments,
  hlsSegmentGroups,
  hlsSourceList,
  hlsSourceCount,
  hlsTotalCount,
  hlsKeptCount,
  hlsRemovedCount,
  hlsKeptDuration,
  hlsDownloadName,
  hlsDefaultDownloadName,
  hlsDownloadExt,
  hlsDownloadDir,
  hlsDownloadDirOptions,
  hlsDownloadConcurrency,
  hlsDownloadParallel,
  saveDownloadParams,
  playHlsDownloadOnline,
  hlsDownloadXcode,
  hlsDownloadList,
  hlsDownloadStats,
  hlsPlayingDownloadId,
  hlsDownloadMeta,
  switchLinkTab,
  submitLink,
  playHlsRemaining,
  removeHlsSegment,
  removeHlsSimilarSegments,
  restoreHlsSegments,
  hlsAdBlacklist,
  hlsAdCount,
  isHlsAdSegment,
  unmarkHlsAdSegments,
  toggleHlsAdSegment,
  clearHlsAdBlacklist,
  removeHlsSource,
  moveHlsSource,
  downloadHls,
  cancelHlsDownload,
  restartHlsDownload,
  downloadHlsInBrowser,
  downloadHlsInBrowserNow,
  browserDownloadInProgress,
  browserNowInProgress,
  browserNowProgressRatio,
  browserProgressRatio,
  chooseHlsDownloadDir,
  playHlsDownload,
  clearDoneHlsDownloads,
  clearFailedHlsDownloads,
  destroyHls,
  cleanup,
} = useLinkPlayback($q, {
  magnetURI: magnetModel,
  submitMagnet: () => props.submitMagnet?.(),
  getVideoEl,
  ensureVideo,
  getVolume,
  onPlay: handlePlay,
  // 服务端可选保存目录：来自系统设置里的媒体目录
  getDownloadDirs: () => systemProperty.getSettingInfo?.Dirs ?? [],
});

/** 下载后转码选项：'' 不转码（默认），其余与后端 hlsAllowedXcode 一致 */
const hlsXcodeOptions = [
  { value: '', label: '不转码', caption: '下载后保持原样（ts / mp4）' },
  { value: 'copy', label: '转 MP4', caption: '仅换封装，速度快' },
  { value: 'h264', label: '转 H264', caption: '重新编码，兼容性最好' },
  { value: 'h265', label: '转 H265', caption: '重新编码，体积更小' },
];

/** 下载并发参数的取值区间（与后端一致，越界由后端兜底截断） */
const DOWNLOAD_PARAM_MIN = 1;
const DOWNLOAD_PARAM_MAX = 16;

/**
 * 数字输入框与整数状态的桥接：输入框始终是字符串，
 * 空值 / 非法值不写入（保留原值），越界夹到区间内。
 */
function numberModel(get: () => number, set: (n: number) => void) {
  return computed({
    get: () => String(get()),
    set: (val: string) => {
      if (String(val).trim() === '') return;
      const n = Math.round(Number(val));
      if (!Number.isFinite(n)) return;
      set(Math.min(DOWNLOAD_PARAM_MAX, Math.max(DOWNLOAD_PARAM_MIN, n)));
    },
  });
}

/** 并发下载数量：单个任务内同时在途的分片数 */
const concurrencyModel = numberModel(
  () => hlsDownloadConcurrency.value,
  (n) => (hlsDownloadConcurrency.value = n),
);
/** 并行任务数量：服务端同时执行的任务数上限 */
const parallelModel = numberModel(
  () => hlsDownloadParallel.value,
  (n) => (hlsDownloadParallel.value = n),
);

/**
 * 下载参数的编辑态：默认只读展示，点「修改参数」进入输入态，
 * 保存（或取消）后回到只读。草稿只在保存时写回，避免输入过程中
 * 影响正在提交的任务。
 */
const paramsEditing = ref(false);
const paramDraft = reactive({ concurrency: '', parallel: '' });

function startEditParams() {
  paramDraft.concurrency = String(hlsDownloadConcurrency.value);
  paramDraft.parallel = String(hlsDownloadParallel.value);
  paramsEditing.value = true;
}

function saveParams() {
  saveDownloadParams(
    Number(paramDraft.concurrency),
    Number(paramDraft.parallel),
  );
  paramsEditing.value = false;
  $q.notify({
    type: 'positive',
    message: `下载参数已保存（并发 ${hlsDownloadConcurrency.value} · 并行 ${hlsDownloadParallel.value}）`,
    position: 'top',
    timeout: 2000,
  });
}

/**
 * 原视频已是 mp4（fMP4 源，产物容器就是 mp4）时不提供转码：
 * 再转一次只是换封装，白白占用任务槽位，服务端也会直接跳过。
 */
const hlsXcodeSkipped = computed(() => hlsDownloadExt.value === 'mp4');

/** 转码下拉按钮文案 */
const hlsDownloadXcodeLabel = computed(() =>
  hlsXcodeSkipped.value
    ? '无需转码'
    : (hlsXcodeOptions.find((item) => item.value === hlsDownloadXcode.value)
        ?.label ?? '下载后转码'),
);

/** 浏览器直下按钮文案：未进行时为固定文案，进行中显示实时百分比 */
const browserNowLabel = computed(() => {
  const r = browserNowProgressRatio();
  return r == null ? '浏览器直下' : `直下中 ${Math.round(r * 100)}%`;
});

// ── 下载列表过滤 ─────────────────────────────────────────────────────────────
/** 列表状态分组，同时决定条目配色：执行中橘色 / 完成绿色 / 失败（含取消）红色 */
type DownloadBucket = 'downloading' | 'done' | 'failed';
/** 当前过滤项：all 为不过滤 */
const hlsDownloadFilter = ref<DownloadBucket | 'all'>('all');

/** 条目归入哪一组：浏览器直下进行中同样算「执行中」 */
function hlsDownloadBucket(item: HlsDownloadItem): DownloadBucket {
  // 排队中与执行中同组：都用「执行中」筛选一起看
  if (
    item.status === 'downloading' ||
    item.status === 'queued' ||
    browserDownloadInProgress(item)
  ) {
    return 'downloading';
  }
  if (item.status === 'done') return 'done';
  return 'failed';
}

/** 依过滤项展示的下载列表 */
const hlsVisibleDownloadList = computed(() =>
  hlsDownloadFilter.value === 'all'
    ? hlsDownloadList.value
    : hlsDownloadList.value.filter(
        (item) => hlsDownloadBucket(item) === hlsDownloadFilter.value,
      ),
);

/** 点表头的分组统计即切换过滤，再点一次回到全部 */
function toggleDownloadFilter(bucket: DownloadBucket | 'all') {
  hlsDownloadFilter.value = hlsDownloadFilter.value === bucket ? 'all' : bucket;
}

/** 条目主色：执行中橘色 / 完成绿色 / 失败红色 */
const downloadBucketColor: Record<DownloadBucket, string> = {
  downloading: 'orange-4',
  done: 'green-4',
  failed: 'red-4',
};

function hlsDownloadIconColor(item: HlsDownloadItem): string {
  return downloadBucketColor[hlsDownloadBucket(item)];
}

// ── 可见的链接类型 ────────────────────────────────────────────────────────────
const visibleTabs = computed(() =>
  LINK_TABS.filter((item) => props.tabs.includes(item.value)),
);

// 宿主未开放上次记住的 tab（如弹窗里没有磁力链）时回退到第一个可用 tab
watch(
  visibleTabs,
  (list) => {
    const first = list[0];
    if (!first || list.some((item) => item.value === linkTab.value)) return;
    switchLinkTab(first.value);
  },
  { immediate: true },
);

// ── 分片链接复制 ──────────────────────────────────────────────────────────────
async function copySegmentUrl(seg: HlsSegment) {
  if (!seg?.url) return;
  await copyText(seg.url);
  $q.notify({
    type: 'positive',
    message: `已复制第 ${seg.index} 个分片链接`,
    position: 'top',
    timeout: 1200,
  });
}

async function copyAllSegmentUrls() {
  const urls = hlsSegments.value
    .filter((seg) => !isHlsAdSegment(seg))
    .map((seg) => seg.url);
  if (urls.length === 0) {
    $q.notify({
      type: 'warning',
      message: '没有可复制的分片链接',
      position: 'top',
    });
    return;
  }
  await copyText(urls.join('\n'));
  $q.notify({
    type: 'positive',
    message: `已复制 ${urls.length} 个分片链接`,
    position: 'top',
    timeout: 1500,
  });
}

// ── 停止播放 / 卸载 ───────────────────────────────────────────────────────────
/**
 * 停止播放并隐藏播放器：清掉播放源、销毁 HLS 实例（停止继续拉分片），
 * 之后播放器区域收起，重新播放或重新解析链接时自动展开。
 */
function stopAndHidePlayer() {
  stopPlayback();
  playerStopped.value = true;
}

/** 销毁 HLS 实例并清空播放地址（宿主切换资源 / 关闭弹窗时调用） */
function stopPlayback() {
  destroyHls();
  const videoEl = internalVideoRef.value;
  if (!videoEl) return;
  try {
    videoEl.pause();
  } catch {
    /* 忽略暂停异常 */
  }
  videoEl.removeAttribute('src');
  videoEl.load();
  embeddedActive.value = false;
}

onBeforeUnmount(() => {
  stopPlayback();
  cleanup();
});

defineExpose({ destroyHls, stopPlayback, cleanup });
</script>

<style scoped lang="scss">
/* 宽度完全交给宿主容器（弹窗 92vw / 页面 64vw），面板自身只按比例铺满 */
.link-source-panel {
  width: 100%;
  margin: 0 auto;
}

/* 播放中：两列网格——左列上半是分片列表、下半是下载列表，右列是播放器。
   列宽按 54 : 46 固定比例；行高按「顶部按内容、剩余空间平分」分配——
   面板高度由宿主容器（弹窗）给定，播放器高度 = 弹窗高度 - 顶部高度 */
.link-source-panel-playing {
  display: grid;
  grid-template-columns: 54fr 46fr;
  grid-template-rows: auto auto minmax(0, 1fr) minmax(0, 1fr);
  column-gap: 10px;
  height: 100%;
}

/* 左列两块撑满各自的行：面板变 flex 列，列表吃掉剩余高度并内部滚动 */
.link-source-panel-playing > .hls-segment-panel,
.link-source-panel-playing > .link-source-stage {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.link-source-panel-playing .hls-download-list-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.link-source-panel-playing .link-source-stage-download {
  flex: 1 1 auto;
  min-height: 0;
}

.link-source-panel-playing .hls-segment-list,
.link-source-panel-playing .hls-download-list {
  flex: 1 1 auto;
  min-height: 0;
  /* 高度交给行比例分配，不再用 vh 限高 */
  max-height: none;
}

/* 标题行保持自身高度，不被列表挤压 */
.link-source-panel-playing .hls-segment-header,
.link-source-panel-playing .hls-download-list-header {
  flex-shrink: 0;
}

/* 顶部区块跨满两列 */
.link-source-panel-playing > .link-tabs,
.link-source-panel-playing > .magnet-input-wrapper {
  grid-column: 1 / -1;
}

/* 其余区块（分片列表、下载列表）落在左列 */
.link-source-panel-playing > * {
  grid-column: 1;
  min-width: 0;
}

/* 播放器占右列：从第 3 行（列表开始处）跨到末尾，
   拉伸满整行高度，内部播放器用 height: 100% 铺满 */
.link-source-panel-playing > .link-source-stage-play {
  grid-column: 2;
  /* 跨分片列表 + 下载列表两行 = 弹窗去掉顶部后的全部高度 */
  grid-row: 3 / span 2;
  align-self: stretch;
  min-width: 0;
  min-height: 0;
}

/* ── 播放区 / 下载区 ────────────────────────────────────────────────────────── */
/* 上下排列：下载列表（左列内）在上，播放器由网格排到右侧 */
.link-source-stage {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 10px;
}

.link-source-stage-play,
.link-source-stage-download {
  min-width: 0;
}

/* 右列播放器：宽度由网格列决定，高度铺满左列列表区（没有上方 12px 间距） */
.link-source-stage-play {
  .link-source-player {
    height: 100%;
    /* 高度由外部区域决定时不再按 16:9 计算，画面靠 object-fit: contain 保持比例 */
    aspect-ratio: auto;
    max-width: 100%;
    max-height: none;
    min-height: 180px;
    margin: 0;
  }
}

/* ── 内嵌播放器（批量编辑弹窗等宿主用） ─────────────────────────────────────── */
.link-source-player {
  position: relative;
  width: 100%;
  /* 高度按 16:9 比例随宽度变化；矮屏时由 max-height 兜底（画面 contain 不变形） */
  height: auto;
  aspect-ratio: 16 / 9;
  max-height: 46vh;
  margin: 12px auto 0;
  background: #06060c;
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 14px;
  overflow: hidden;
}

.link-source-video {
  display: block;
  width: 100%;
  height: 100%;
  /* 播放器高度跟随左列，画面按比例缩放、留黑边而不拉伸变形 */
  object-fit: contain;
  background: #000;
}

/* 停止按钮：浮在播放器右上角，避免被原生控制条遮挡 */
.link-source-player-stop {
  position: absolute;
  top: 6px;
  right: 6px;
  z-index: 2;
  background: rgba(8, 8, 16, 0.72);
}

/* ── 链接类型 Tab ──────────────────────────────────────────────────────────── */
.link-tabs {
  display: flex;
  justify-content: center;
  gap: 6px;
  margin-bottom: 10px;
}

.link-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 14px;
  font-family: inherit;
  font-size: 0.78rem;
  white-space: nowrap;
  color: rgba(196, 181, 253, 0.7);
  background: rgba(12, 12, 24, 0.6);
  border: 1px solid rgba(99, 102, 241, 0.22);
  border-radius: 20px;
  cursor: pointer;
  transition:
    color 0.2s,
    background 0.2s,
    border-color 0.2s,
    box-shadow 0.2s;
}

.link-tab:hover {
  color: #e0e7ff;
  background: rgba(99, 102, 241, 0.16);
  border-color: rgba(99, 102, 241, 0.45);
}

.link-tab-active {
  color: #fff;
  background: linear-gradient(
    135deg,
    rgba(99, 102, 241, 0.5),
    rgba(139, 92, 246, 0.45)
  );
  border-color: rgba(139, 92, 246, 0.7);
  box-shadow: 0 0 14px rgba(99, 102, 241, 0.35);
}

/* ── 链接输入框 ────────────────────────────────────────────────────────────── */
.magnet-input-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(12, 12, 24, 0.75);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 40px;
  padding: 8px 8px 8px 18px;
  transition:
    border-color 0.3s,
    box-shadow 0.3s;
}

.magnet-input-wrapper.magnet-focused {
  border-color: rgba(139, 92, 246, 0.65);
  box-shadow:
    0 0 0 3px rgba(99, 102, 241, 0.12),
    0 0 30px rgba(99, 102, 241, 0.18);
}

.magnet-icon {
  flex-shrink: 0;
  opacity: 0.8;
}

.magnet-input {
  flex: 1;
}

.magnet-input :deep(.q-field__control) {
  background: transparent;
  border: none;
}

.magnet-input :deep(.q-field__native) {
  color: #c4b5fd;
  font-size: 0.88rem;
}

.magnet-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.38);
}

/* 分片链接多行输入：胶囊形改圆角矩形，图标 / 按钮与首行对齐 */
.magnet-input-wrapper.magnet-input-multi {
  align-items: flex-start;
  border-radius: 18px;
  padding: 10px 8px 10px 14px;
}

.magnet-input-wrapper.magnet-input-multi .magnet-icon {
  margin-top: 6px;
}

.magnet-input-wrapper.magnet-input-multi .magnet-submit-btn {
  margin-top: 2px;
}

.hls-url-rows {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  /* 地址较多时内部滚动，输入框高度不无限增长 */
  max-height: 112px;
  overflow-y: auto;
  padding: 2px 0;
}

.hls-url-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.hls-url-field {
  flex: 1;
  min-width: 0;
}

.hls-url-field :deep(.q-field__control) {
  background: transparent;
  border: none;
  min-height: 26px;
}

.hls-url-field :deep(.q-field__native) {
  color: #c4b5fd;
  font-size: 0.86rem;
}

.hls-url-field :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.38);
}

.hls-url-clear-btn {
  flex-shrink: 0;
  margin-top: 2px;
  opacity: 0.75;
}

.magnet-submit-btn {
  flex-shrink: 0;
  padding: 0 14px;
  border-radius: 24px;
  background: rgba(99, 102, 241, 0.2);
  border: 1px solid rgba(99, 102, 241, 0.4);
  transition:
    background 0.2s,
    box-shadow 0.2s;
}

.magnet-submit-btn:hover:not([disabled]) {
  background: rgba(99, 102, 241, 0.4);
  box-shadow: 0 0 16px rgba(99, 102, 241, 0.4);
}

/* ── 分片列表（解析后可删除广告分片） ──────────────────────────────────────── */
.hls-segment-panel {
  margin-top: 10px;
  background: rgba(12, 12, 24, 0.85);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 14px;
  overflow: hidden;
}

.hls-segment-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
}

/* 下载设置行（文件名 + 目录 + 下载按钮）：独立一行，文件名自适应剩余宽度以撑满整行 */
.hls-download-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
  background: rgba(99, 102, 241, 0.05);
}

.hls-play-btn {
  flex-shrink: 0;
  min-height: 24px;
  padding: 0 10px;
  font-size: 0.74rem;
}

.hls-segment-summary {
  font-size: 0.76rem;
  color: rgba(196, 181, 253, 0.85);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 自定义下载文件名输入框：独占剩余宽度，把同行按钮顶到右边 */
.hls-filename-input {
  flex: 1 1 auto;
  min-width: 120px;
}

/* 下载按钮：固定宽度不参与压缩，保证与目录/文件名同排 */
.hls-download-btn {
  flex-shrink: 0;
  min-height: 24px;
  padding: 0 10px;
  font-size: 0.74rem;
}

/* 「下载目录」按钮：目录名可能较长，超出省略 */
.hls-dir-btn {
  flex-shrink: 0;
  min-height: 24px;
  max-width: 150px;
  padding: 0 8px;
  font-size: 0.74rem;
}

.hls-dir-btn :deep(.q-btn__content) {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

/* 下载目录下拉菜单：路径可能较长，限制宽度并允许折行 */
.hls-dir-menu {
  max-width: 320px;
  max-height: 260px;
  overflow-y: auto;
}

.hls-dir-menu-label {
  word-break: break-all;
  font-size: 0.76rem;
}

/* 下载后转码下拉：菜单宽度贴合内容，按钮内文案不换行 */
.hls-xcode-btn :deep(.q-btn__content) {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.hls-xcode-menu {
  min-width: 180px;
}

/* 广告黑名单菜单：URL 前缀较长，限宽并允许折行 */
.hls-ad-menu {
  max-width: 360px;
  max-height: 300px;
  overflow-y: auto;
}

.hls-ad-menu-label {
  word-break: break-all;
  font-size: 0.76rem;
}

.hls-filename-input :deep(.q-field__control) {
  height: 26px;
  min-height: 26px;
  padding: 0 6px;
  border-radius: 8px;
  background: rgba(99, 102, 241, 0.12);
}

.hls-filename-input :deep(.q-field__native) {
  font-size: 0.74rem;
  color: rgba(224, 231, 255, 0.95);
  padding: 0;
}

.hls-filename-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.45);
}

/* 并发 / 并行参数：固定窄宽，避免挤占文件名输入的剩余空间 */
.hls-param-input {
  flex: 0 0 auto;
  width: 92px;
}

.hls-param-input :deep(.q-field__control) {
  height: 26px;
  min-height: 26px;
  padding: 0 6px;
  border-radius: 8px;
  background: rgba(99, 102, 241, 0.12);
}

.hls-param-input :deep(.q-field__native) {
  font-size: 0.74rem;
  color: rgba(224, 231, 255, 0.95);
  padding: 0;
  text-align: center;
}

.hls-param-input :deep(.q-field__prefix) {
  font-size: 0.72rem;
  color: rgba(165, 148, 249, 0.75);
  padding-right: 4px;
}

/* 数字输入框自带的步进箭头在窄输入框里很挤，隐藏掉（仍可键盘输入） */
.hls-param-input :deep(input[type='number']::-webkit-outer-spin-button),
.hls-param-input :deep(input[type='number']::-webkit-inner-spin-button) {
  -webkit-appearance: none;
  margin: 0;
}

.hls-filename-input :deep(.q-field__prepend) {
  padding-right: 4px;
}

.hls-segment-list {
  /* 按视口比例限高，条目多时在列表内部滚动（不撑高弹窗） */
  max-height: 22vh;
  overflow-y: auto;
  padding: 4px 0;
}

/* ── 源列表（多个 m3u8 的合并顺序） ────────────────────────────────────────── */
.hls-source-list {
  max-height: 12vh;
  overflow-y: auto;
  padding: 4px 0;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
  background: rgba(139, 92, 246, 0.06);
}

.hls-source-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 10px;
  font-size: 0.74rem;
  color: rgba(196, 181, 253, 0.75);
  transition: background 0.15s;
}

.hls-source-item:hover {
  background: rgba(139, 92, 246, 0.14);
}

.hls-source-order,
.hls-url-order {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  font-size: 0.66rem;
  color: #e0e7ff;
  background: rgba(99, 102, 241, 0.35);
  border-radius: 50%;
  font-variant-numeric: tabular-nums;
}

.hls-source-url {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
}

.hls-source-meta {
  flex-shrink: 0;
  color: rgba(165, 148, 249, 0.8);
  font-variant-numeric: tabular-nums;
}

/* ── 分片列表里的源分组标题 ────────────────────────────────────────────────── */
.hls-segment-group {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding: 2px 10px;
  font-size: 0.68rem;
  color: rgba(148, 163, 184, 0.9);
  background: rgba(99, 102, 241, 0.08);
}

.hls-segment-group-label {
  flex-shrink: 0;
  color: rgba(129, 140, 248, 0.95);
}

.hls-segment-group-url {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
}

.hls-segment-group-count {
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.hls-segment-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 3px 10px;
  font-size: 0.74rem;
  color: rgba(196, 181, 253, 0.75);
  transition: background 0.15s;
}

.hls-segment-item:hover {
  background: rgba(99, 102, 241, 0.12);
}

/* 广告（黑名单）分片：仍在列表中，但置灰 + 未勾选，表示不参与播放 / 下载 */
.hls-segment-item--ad {
  opacity: 0.45;
  filter: grayscale(1);
}

.hls-segment-item--ad .hls-segment-url {
  text-decoration: line-through;
  cursor: default;
}

.hls-segment-item--ad:hover {
  background: rgba(148, 163, 184, 0.1);
}

/* 勾选状态图标：不参与下载的分片显示未勾选框 */
.hls-segment-check {
  flex-shrink: 0;
}

.hls-segment-index {
  flex-shrink: 0;
  width: 42px;
  color: rgba(129, 140, 248, 0.7);
  font-variant-numeric: tabular-nums;
}

.hls-segment-duration {
  flex-shrink: 0;
  width: 52px;
  color: rgba(165, 148, 249, 0.8);
  font-variant-numeric: tabular-nums;
}

.hls-segment-url {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
  cursor: pointer;
  transition: color 0.15s;
}

.hls-segment-url:hover {
  color: rgba(165, 180, 252, 1);
  text-decoration: underline;
}

/* ── 下载列表（已完成的分片视频，可回放） ──────────────────────────────────── */
/* 下载列表已是独立卡片（不再嵌在分片面板内），底色必须自给自足：
   否则会直接透出宿主的浅色页面背景（批量编辑弹窗是 bg-grey-4） */
.hls-download-list-panel {
  background: rgba(12, 12, 24, 0.85);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 14px;
  overflow: hidden;
}

.hls-download-list-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.16);
}

.hls-download-list-title {
  font-size: 0.76rem;
  color: rgba(134, 239, 172, 0.9);
  white-space: nowrap;
}

/* 下载列表统计：执行中 / 完成 / 失败 */
.hls-download-stat {
  font-size: 0.7rem;
  padding: 1px 6px;
  border-radius: 8px;
  white-space: nowrap;
  background: rgba(148, 163, 184, 0.16);
  color: rgba(226, 232, 240, 0.85);
}

.hls-download-stat-running {
  background: rgba(96, 165, 250, 0.18);
  color: rgba(147, 197, 253, 0.95);
}

.hls-download-stat-done {
  background: rgba(74, 222, 128, 0.16);
  color: rgba(134, 239, 172, 0.95);
}

.hls-download-stat-failed {
  background: rgba(248, 113, 113, 0.18);
  color: rgba(252, 165, 165, 0.95);
}

/* 队列中：等待空闲槽位，用中性色与「执行中」区分 */
.hls-download-stat-queued {
  background: rgba(148, 163, 184, 0.18);
  color: rgba(203, 213, 225, 0.95);
}

/* 下载参数：只读文案 / 编辑态输入，紧跟在列表状态后面 */
.hls-download-params {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.72rem;
  color: rgba(165, 148, 249, 0.9);
  white-space: nowrap;
}

/* 统计可点击筛选：悬停有反馈，选中项加描边 */
.hls-download-stat-btn {
  cursor: pointer;
  user-select: none;
  transition: box-shadow 0.15s, filter 0.15s;
}

.hls-download-stat-btn:hover {
  filter: brightness(1.25);
}

.hls-download-stat-active {
  box-shadow: 0 0 0 1px currentColor inset;
  filter: brightness(1.2);
}

.hls-download-list {
  max-height: 28vh;
  overflow-y: auto;
  padding: 4px 0;
}

.hls-download-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  font-size: 0.74rem;
  color: rgba(196, 181, 253, 0.8);
  transition: background 0.15s;
}

.hls-download-item:hover {
  background: rgba(34, 197, 94, 0.1);
}

/* 当前正在回放的条目：高亮提示，避免误播 */
.hls-download-item-playing {
  background: rgba(34, 197, 94, 0.14);
}

/* 条目按任务状态着色：执行中橘 / 完成绿 / 失败红 */
.hls-download-item-downloading {
  border-left: 2px solid rgba(251, 146, 60, 0.85);
  background: rgba(251, 146, 60, 0.08);
}

.hls-download-item-done {
  border-left: 2px solid rgba(74, 222, 128, 0.75);
  background: rgba(74, 222, 128, 0.06);
}

.hls-download-item-failed {
  border-left: 2px solid rgba(248, 113, 113, 0.85);
  background: rgba(248, 113, 113, 0.08);
}

.hls-download-item-downloading .hls-download-item-meta {
  color: rgba(253, 186, 116, 0.9);
}

.hls-download-item-done .hls-download-item-meta {
  color: rgba(134, 239, 172, 0.9);
}

.hls-download-item-failed .hls-download-item-meta {
  color: rgba(252, 165, 165, 0.9);
}

.hls-download-item-icon {
  flex-shrink: 0;
}

.hls-download-item-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.hls-download-item-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: rgba(224, 231, 255, 0.95);
}

.hls-download-item-meta {
  font-size: 0.68rem;
  color: rgba(148, 163, 184, 0.85);
  font-variant-numeric: tabular-nums;
}

/* 下载中的进度条：贴在文件名 / 进度文本下方 */
.hls-download-item-bar {
  margin-top: 3px;
  border-radius: 2px;
}

/* 容器不够宽时左右布局会挤压两侧，回落到上下排列 */
@media (max-width: 900px) {
  /* 不够宽就回落单列，仍然是比例划分 */
  .link-source-panel-playing {
    grid-template-columns: 1fr;
  }

  .link-source-panel-playing > .link-source-stage-play {
    grid-column: 1;
    grid-row: auto;
    width: 100%;
  }

  /* 单列时没有可拉伸的行高，给播放器一个按视口比例的固定高度 */
  .link-source-stage-play .link-source-player {
    height: 38vh;
  }
}

@media (max-width: 768px) {
  .link-source-player {
    height: 34vh;
  }

  .hls-segment-list {
    max-height: 20vh;
  }

  /* 小屏下源列表同样收窄，避免挤掉分片列表 */
  .hls-source-list {
    max-height: 10vh;
  }

  .hls-url-rows {
    max-height: 10vh;
  }

  .hls-segment-url {
    display: none;
  }

  .link-tabs {
    gap: 4px;
  }

  .link-tab {
    padding: 4px 10px;
    font-size: 0.72rem;
  }
}
</style>
