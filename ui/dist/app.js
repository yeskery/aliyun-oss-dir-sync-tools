// 阿里云 OSS 双 Bucket 可视化文件管理工作台
(function () {
  'use strict';

  // 全局工作区状态
  let currentDir = '';
  let currentListing = null;
  let systemStatus = null;
  let dragCounter = 0;

  // DOM 元素缓存
  const els = {
    // 顶部工具栏
    statusBadge: document.getElementById('status-badge'),
    btnSettings: document.getElementById('btn-settings'),
    btnRefresh: document.getElementById('btn-refresh'),
    btnCopyCurrDir: document.getElementById('btn-copy-curr-dir'),
    btnRefreshDir: document.getElementById('btn-refresh-dir'),
    btnDeleteCurrDir: document.getElementById('btn-delete-curr-dir'),
    btnNewDir: document.getElementById('btn-new-dir'),
    btnUpload: document.getElementById('btn-upload'),
    btnUploadFolder: document.getElementById('btn-upload-folder'),
    fileInput: document.getElementById('file-input'),
    folderInput: document.getElementById('folder-input'),
    searchInput: document.getElementById('search-input'),
    btnViewGrid: document.getElementById('btn-view-grid'),
    btnViewList: document.getElementById('btn-view-list'),

    // 导航区
    breadcrumbs: document.getElementById('breadcrumbs'),

    // 工作区与拖拽
    workspaceArea: document.getElementById('workspace-area'),
    dropOverlay: document.getElementById('drop-overlay'),
    dropTargetName: document.getElementById('drop-target-name'),
    foldersSection: document.getElementById('folders-section'),
    foldersGrid: document.getElementById('folders-grid'),
    filesSection: document.getElementById('files-section'),
    filesSectionLabel: document.getElementById('files-section-label'),
    filesGrid: document.getElementById('files-grid'),
    filesListContainer: document.getElementById('files-list-container'),
    filesTableBody: document.getElementById('files-table-body'),
    emptyState: document.getElementById('empty-state'),

    // 设置弹窗
    modalConfig: document.getElementById('modal-config'),
    btnCloseConfig: document.getElementById('btn-close-config'),
    btnCancelConfig: document.getElementById('btn-cancel-config'),
    btnSaveConfig: document.getElementById('btn-save-config'),
    btnCopyFromA: document.getElementById('btn-copy-from-a'),

    // Bucket A
    tabsA: document.getElementById('tabs-a'),
    paneAAk: document.getElementById('pane-a-ak'),
    paneAAccount: document.getElementById('pane-a-account'),
    paneACode: document.getElementById('pane-a-code'),
    akA: document.getElementById('ak-a'),
    skA: document.getElementById('sk-a'),
    endpointA: document.getElementById('endpoint-a'),
    bucketA: document.getElementById('bucket-a'),
    cdnDomainA: document.getElementById('cdn-domain-a'),
    btnTestA: document.getElementById('btn-test-a'),
    feedbackA: document.getElementById('feedback-a'),
    toggleSkA: document.getElementById('toggle-sk-a'),
    btnOpenLoginA: document.getElementById('btn-open-login-a'),
    accBadgeA: document.getElementById('account-status-badge-a'),
    accMsgA: document.getElementById('account-status-msg-a'),
    endpointAAcc: document.getElementById('endpoint-a-acc'),
    bucketAAcc: document.getElementById('bucket-a-acc'),
    btnListBucketsA: document.getElementById('btn-list-buckets-a'),
    bucketSelectA: document.getElementById('bucket-select-a'),
    authcodeInputA: document.getElementById('authcode-input-a'),
    btnParseCodeA: document.getElementById('btn-parse-code-a'),
    endpointACode: document.getElementById('endpoint-a-code'),
    bucketACode: document.getElementById('bucket-a-code'),

    // Bucket B
    tabsB: document.getElementById('tabs-b'),
    paneBAk: document.getElementById('pane-b-ak'),
    paneBAccount: document.getElementById('pane-b-account'),
    paneBCode: document.getElementById('pane-b-code'),
    akB: document.getElementById('ak-b'),
    skB: document.getElementById('sk-b'),
    endpointB: document.getElementById('endpoint-b'),
    bucketB: document.getElementById('bucket-b'),
    cdnDomainB: document.getElementById('cdn-domain-b'),
    btnTestB: document.getElementById('btn-test-b'),
    feedbackB: document.getElementById('feedback-b'),
    toggleSkB: document.getElementById('toggle-sk-b'),
    btnOpenLoginB: document.getElementById('btn-open-login-b'),
    accBadgeB: document.getElementById('account-status-badge-b'),
    accMsgB: document.getElementById('account-status-msg-b'),
    endpointBAcc: document.getElementById('endpoint-b-acc'),
    bucketBAcc: document.getElementById('bucket-b-acc'),
    btnListBucketsB: document.getElementById('btn-list-buckets-b'),
    bucketSelectB: document.getElementById('bucket-select-b'),
    authcodeInputB: document.getElementById('authcode-input-b'),
    btnParseCodeB: document.getElementById('btn-parse-code-b'),
    endpointBCode: document.getElementById('endpoint-b-code'),
    bucketBCode: document.getElementById('bucket-b-code'),

    syncDirsBox: document.getElementById('sync-dirs-box'),
    newDirInput: document.getElementById('new-sync-dir-input'),
    btnAddSyncDir: document.getElementById('btn-add-sync-dir'),

    // 凭据过期重登横幅
    reloginBanner: document.getElementById('relogin-banner'),
    reloginMsg: document.getElementById('relogin-msg'),
    bannerBtnLoginA: document.getElementById('banner-btn-login-a'),
    bannerBtnLoginB: document.getElementById('banner-btn-login-b'),
    bannerBtnSettings: document.getElementById('banner-btn-settings'),
    bannerBtnClose: document.getElementById('banner-btn-close'),

    // ACL 读写权限弹窗
    modalAcl: document.getElementById('modal-acl'),
    btnCloseAcl: document.getElementById('btn-close-acl'),
    btnCancelAcl: document.getElementById('btn-cancel-acl'),
    btnConfirmAcl: document.getElementById('btn-confirm-acl'),
    aclFileName: document.getElementById('acl-file-name'),
    aclBtnGroup: document.getElementById('acl-btn-group'),
    aclDesc: document.getElementById('acl-desc'),

    // 图片大图预览弹窗
    modalPreview: document.getElementById('modal-preview'),
    btnClosePreview: document.getElementById('btn-close-preview'),
    previewImg: document.getElementById('preview-img'),
    previewTitle: document.getElementById('preview-title'),

    // 复制外网链接弹窗
    modalCopyUrl: document.getElementById('modal-copy-url'),
    btnCloseCopyUrl: document.getElementById('btn-close-copy-url'),
    btnCancelCopyUrl: document.getElementById('btn-cancel-copy-url'),
    copyUrlFileName: document.getElementById('copy-url-file-name'),
    copyUrlOptionsList: document.getElementById('copy-url-options-list'),
    copyUrlExpiresInput: document.getElementById('copy-url-expires-input'),
    btnGenerateSignedUrl: document.getElementById('btn-generate-signed-url'),

    // 差异同步弹窗
    modalDiffSync: document.getElementById('modal-diff-sync'),
    btnCloseDiffSync: document.getElementById('btn-close-diff-sync'),
    btnCancelDiffSync: document.getElementById('btn-cancel-diff-sync'),
    diffSyncFileName: document.getElementById('diff-sync-file-name'),
    diffBnameA: document.getElementById('diff-bname-a'),
    diffSizeA: document.getElementById('diff-size-a'),
    diffTimeA: document.getElementById('diff-time-a'),
    diffEtagA: document.getElementById('diff-etag-a'),
    diffBnameB: document.getElementById('diff-bname-b'),
    diffSizeB: document.getElementById('diff-size-b'),
    diffTimeB: document.getElementById('diff-time-b'),
    diffEtagB: document.getElementById('diff-etag-b'),
    btnDiffSyncA: document.getElementById('btn-diff-sync-a'),
    btnDiffSyncB: document.getElementById('btn-diff-sync-b'),

    // 新建目录弹窗
    modalNewDir: document.getElementById('modal-new-dir'),
    btnCloseNewDir: document.getElementById('btn-close-new-dir'),
    btnCancelNewDir: document.getElementById('btn-cancel-new-dir'),
    btnConfirmNewDir: document.getElementById('btn-confirm-new-dir'),
    newDirParentPath: document.getElementById('new-dir-parent-path'),
    inputNewDirName: document.getElementById('input-new-dir-name'),

    // 彻底删除文件/文件夹弹窗
    modalConfirmDelete: document.getElementById('modal-confirm-delete'),
    btnCloseDeleteModal: document.getElementById('btn-close-delete-modal'),
    btnCancelDeleteModal: document.getElementById('btn-cancel-delete-modal'),
    btnConfirmDeleteAction: document.getElementById('btn-confirm-delete-action'),
    deleteModalTitle: document.getElementById('delete-modal-title'),
    deleteWarningText: document.getElementById('delete-warning-text'),
    deleteInfoTypeLabel: document.getElementById('delete-info-type-label'),
    deleteTargetName: document.getElementById('delete-target-name'),
    deleteTargetKey: document.getElementById('delete-target-key'),

    // Toast
    toast: document.getElementById('toast')
  };

  let currentViewMode = localStorage.getItem('oss_sync_view_mode') || 'grid'; // 'grid' | 'list'
  let currentAclTarget = { key: '', name: '', acl: 'default' };

  const aclDescriptions = {
    'default': '描述  文件ACL权限将继承所在Bucket的ACL权限',
    'private': '描述  只有该Bucket的拥有者可以对该Object进行读写操作，其他人均无权限',
    'public-read': '描述  任何人（包括匿名访问者）都对该Object具有读权限，但只有Bucket拥有者具有写权限',
    'public-read-write': '描述  任何人（包括匿名访问者）都对该Object具有读权限和写权限，可能产生较高费用，请谨慎设置'
  };

  let authState = {
    a: {
      authType: 'ak', // 'ak' | 'account' | 'code'
      ak: '',
      sk: '',
      securityToken: '',
      endpoint: 'oss-cn-hangzhou.aliyuncs.com',
      bucket: '',
      pollTimer: null
    },
    b: {
      authType: 'ak',
      ak: '',
      sk: '',
      securityToken: '',
      endpoint: 'oss-cn-hangzhou.aliyuncs.com',
      bucket: '',
      pollTimer: null
    }
  };

  let editConfig = {
    bucket_a: {},
    bucket_b: {},
    sync_dirs: []
  };

  // Toast 提示
  function showToast(msg, type = 'info') {
    els.toast.textContent = msg;
    els.toast.className = `toast toast-${type} show`;
    clearTimeout(els.toast._t);
    els.toast._t = setTimeout(() => {
      els.toast.classList.remove('show');
    }, 3500);
  }

  // 检测阿里云访问凭证是否失效或过期 (STS Token 有效期约 6 小时)
  function checkIfTokenExpired(msg) {
    if (!msg || typeof msg !== 'string') return false;
    const lower = msg.toLowerCase();
    const isExpired = 
      lower.includes('securitytokenexpired') ||
      lower.includes('invalidaccesskeyid') ||
      lower.includes('token has expired') ||
      lower.includes('token is expired') ||
      lower.includes('signaturedoesnotmatch') ||
      lower.includes('accessdenied') ||
      lower.includes('the access key id does not exist') ||
      lower.includes('expired') ||
      lower.includes('过期') ||
      lower.includes('失效') ||
      lower.includes('鉴权失败');

    if (isExpired) {
      if (systemStatus) {
        systemStatus.auth_expired = true;
        systemStatus.auth_message = msg;
      }
      updateStatusBadge();
      showReloginBanner(msg);
      return true;
    }
    return false;
  }

  function showReloginBanner(reason) {
    if (!els.reloginBanner) return;
    if (reason && els.reloginMsg) {
      let snippet = reason;
      if (snippet.length > 70) snippet = snippet.substring(0, 70) + '...';
      els.reloginMsg.textContent = `阿里云访问凭据异常 (${snippet})，请点击右侧按钮重新登录以刷新凭证`;
    }
    els.reloginBanner.style.display = 'flex';
  }

  function hideReloginBanner() {
    if (els.reloginBanner) {
      els.reloginBanner.style.display = 'none';
    }
  }

  // 格式化文件大小
  function formatSize(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i];
  }

  // 格式化时间
  function formatTime(isoStr) {
    if (!isoStr || isoStr.startsWith('0001')) return '-';
    const d = new Date(isoStr);
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }
  function pad(n) { return n < 10 ? '0' + n : n; }

  // 1. 初始化检查状态与配置
  async function checkSystemStatus() {
    try {
      const res = await fetch('/api/status');
      const data = await res.json();
      if (data.success && data.data) {
        systemStatus = data.data;
        updateStatusBadge();

        if (!systemStatus.configured) {
          // 未配置则弹出引导
          openConfigModal();
        } else {
          // 已配置则加载根目录
          loadDirectory('');
        }
      }
    } catch (e) {
      console.error('检查状态异常:', e);
      showToast('连接本地服务异常', 'error');
    }
  }

  function updateStatusBadge() {
    if (!systemStatus || !systemStatus.configured) {
      els.statusBadge.className = 'status-badge status-DIFF';
      els.statusBadge.textContent = '🟠 存储桶待配置';
      els.statusBadge.style.cursor = 'pointer';
      els.statusBadge.onclick = openConfigModal;
      return;
    }

    if (systemStatus.auth_expired) {
      els.statusBadge.className = 'status-badge status-DIFF';
      els.statusBadge.textContent = '🔴 凭据已失效 (点击重新登录)';
      els.statusBadge.title = systemStatus.auth_message || '阿里云访问凭证已失效，点击打开配置重新登录';
      els.statusBadge.style.cursor = 'pointer';
      els.statusBadge.onclick = openConfigModal;
      showReloginBanner(systemStatus.auth_message);
      return;
    }

    const aName = systemStatus.bucket_a.bucket_name || 'A';
    const bName = systemStatus.bucket_b.bucket_name || 'B';
    els.statusBadge.className = 'status-badge status-SYNCED';
    els.statusBadge.textContent = `🟢 已连接: ${aName} ⇋ ${bName}`;
    els.statusBadge.title = '双端存储桶已连接，点击可管理配置';
    els.statusBadge.style.cursor = 'pointer';
    els.statusBadge.onclick = openConfigModal;
  }

  // 旋转刷新图标工具函数
  function setRefreshSpinning(spinning) {
    document.querySelectorAll('.refresh-icon').forEach(icon => {
      if (spinning) {
        icon.classList.add('spinning');
      } else {
        icon.classList.remove('spinning');
      }
    });
    if (els.btnRefresh) els.btnRefresh.disabled = spinning;
    if (els.btnRefreshDir) els.btnRefreshDir.disabled = spinning;
  }

  // 2. 加载目录内容
  async function loadDirectory(dir, isManualRefresh = false) {
    currentDir = dir;
    if (isManualRefresh) {
      setRefreshSpinning(true);
    }
    try {
      const res = await fetch(`/api/files?dir=${encodeURIComponent(dir)}&_t=${Date.now()}`);
      const data = await res.json();
      if (data.success && data.data) {
        currentListing = data.data;
        renderListing(currentListing);
        hideReloginBanner();
        if (isManualRefresh) {
          showToast(`已刷新当前目录${currentDir ? '：' + currentDir : ''}`, 'info');
        }
      } else {
        checkIfTokenExpired(data.message);
        if (isManualRefresh) {
          showToast(data.message || '加载目录失败', 'error');
        }
      }
    } catch (e) {
      checkIfTokenExpired(e.message);
      if (isManualRefresh) {
        showToast('加载文件失败: ' + e.message, 'error');
      }
    } finally {
      if (isManualRefresh) {
        setRefreshSpinning(false);
      }
    }
  }

  // 判断指定目录是否为配置中不可直接删除的根同步目录
  function isRootSyncDir(dir) {
    if (!dir) return true;
    const clean = dir.replace(/^[/\\]+|[/\\]+$/g, '');
    if (!clean) return true;
    if (systemStatus && Array.isArray(systemStatus.sync_dirs)) {
      return systemStatus.sync_dirs.some(d => {
        const cd = (d || '').replace(/^[/\\]+|[/\\]+$/g, '');
        return cd === clean;
      });
    }
    return false;
  }

  // 3. 渲染目录与文件
  function renderListing(listing) {
    if (!listing) return;

    // 渲染面包屑
    renderBreadcrumbs(Array.isArray(listing.breadcrumbs) ? listing.breadcrumbs : []);

    // 搜索关键字过滤
    const filterText = (els.searchInput.value || '').trim().toLowerCase();

    // 根目录判断 (兼容 is_root 与 isRoot)
    const isRoot = listing.is_root !== undefined ? listing.is_root : (listing.isRoot || false);

    // 控制当前目录复制路径按钮的状态
    if (els.btnCopyCurrDir) {
      if (currentDir) {
        els.btnCopyCurrDir.disabled = false;
        els.btnCopyCurrDir.title = `复制当前目录路径: ${currentDir}`;
        els.btnCopyCurrDir.style.opacity = '1';
        els.btnCopyCurrDir.style.cursor = 'pointer';
      } else {
        els.btnCopyCurrDir.disabled = true;
        els.btnCopyCurrDir.title = '当前位于根目录，无路径可复制';
        els.btnCopyCurrDir.style.opacity = '0.5';
        els.btnCopyCurrDir.style.cursor = 'not-allowed';
      }
    }

    // 控制当前目录删除按钮的显隐：只有进入非配置白名单根目录的子目录时才展示
    if (els.btnDeleteCurrDir) {
      if (!isRoot && currentDir && !isRootSyncDir(currentDir)) {
        els.btnDeleteCurrDir.style.display = 'inline-flex';
      } else {
        els.btnDeleteCurrDir.style.display = 'none';
      }
    }

    // 过滤子目录与文件
    const rawSubDirs = Array.isArray(listing.sub_dirs) ? listing.sub_dirs : [];
    const subDirs = rawSubDirs.filter(d => d && (!filterText || (d.name && d.name.toLowerCase().includes(filterText))));

    const rawFiles = Array.isArray(listing.files) ? listing.files : [];
    const files = rawFiles.filter(f => f && (!filterText || (f.name && f.name.toLowerCase().includes(filterText))));

    const hasItems = subDirs.length > 0 || files.length > 0;

    // 空状态提示
    if (!hasItems) {
      els.foldersSection.style.display = 'none';
      els.foldersGrid.innerHTML = '';
      els.filesSection.style.display = 'none';
      els.filesGrid.innerHTML = '';
      els.filesTableBody.innerHTML = '';
      els.emptyState.style.display = 'flex';
      if (isRoot) {
        els.emptyState.innerHTML = `
          <span class="empty-state-icon">📂</span>
          <h3>尚未配置同步目录</h3>
          <p>点击右上角「⚙️ 存储桶设置」添加需要同步的目录（例如 images/）</p>
        `;
      } else {
        els.emptyState.innerHTML = `
          <span class="empty-state-icon">📥</span>
          <h3>当前目录为空</h3>
          <p>可直接从电脑中<strong>拖拽图片或文件到此处</strong>立即双端上传并覆盖！</p>
        `;
      }
      return;
    }

    els.emptyState.style.display = 'none';

    if (currentViewMode === 'grid') {
      // ==================== 网格模式 (GRID VIEW) ====================
      // 1. 渲染子目录卡片网格
      if (subDirs.length > 0) {
        els.foldersSection.style.display = 'flex';
        els.foldersGrid.style.display = 'grid';
        els.foldersGrid.innerHTML = '';
        subDirs.forEach(sub => {
          const canDelete = !isRoot && !isRootSyncDir(sub.path);
          const card = document.createElement('div');
          card.className = 'folder-card';
          card.innerHTML = `
            <span class="folder-icon">📁</span>
            <div class="folder-info">
              <span class="folder-name" title="${escapeHtml(sub.name)}">${escapeHtml(sub.name)}</span>
              <span class="folder-hint">${isRoot ? '配置白名单根目录' : '子文件夹'}</span>
            </div>
            ${canDelete ? `
              <div class="folder-actions">
                <button type="button" class="folder-delete-btn" title="彻底删除此文件夹 (双端同步)">🗑️</button>
              </div>
            ` : ''}
          `;
          card.addEventListener('click', () => {
            loadDirectory(sub.path);
          });
          if (canDelete) {
            const delBtn = card.querySelector('.folder-delete-btn');
            if (delBtn) {
              delBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                confirmDeleteDirectory(sub.path, sub.name);
              });
            }
          }
          els.foldersGrid.appendChild(card);
        });
      } else {
        els.foldersSection.style.display = 'none';
        els.foldersGrid.innerHTML = '';
      }

      // 2. 渲染文件卡片网格
      if (files.length > 0) {
        els.filesSection.style.display = 'flex';
        if (els.filesSectionLabel) {
          els.filesSectionLabel.textContent = '📄 文件列表 (FILES)';
        }
        els.filesGrid.style.display = 'grid';
        els.filesGrid.innerHTML = '';
        els.filesListContainer.style.display = 'none';
        els.filesTableBody.innerHTML = '';

        files.forEach(file => {
          const card = document.createElement('div');
          card.className = 'file-card';

          // 缩略图区域
          let thumbHtml = '';
          if (file.is_image) {
            thumbHtml = `
              <div class="file-thumb" data-key="${escapeHtml(file.key)}">
                <img src="/api/preview?key=${encodeURIComponent(file.key)}" alt="${escapeHtml(file.name)}" loading="lazy">
              </div>
            `;
          } else {
            thumbHtml = `
              <div class="file-thumb">
                <span class="file-icon-placeholder">📄</span>
              </div>
            `;
          }

          // 状态徽章与同步按钮
          let statusBadge = '';
          let syncActionBtn = '';
          switch (file.status) {
            case 'SYNCED':
              statusBadge = '<span class="status-badge status-SYNCED">🟢 双端一致</span>';
              break;
            case 'ONLY_A':
              statusBadge = '<span class="status-badge status-ONLY_A">🔵 仅 A 端</span>';
              syncActionBtn = `<button class="action-btn" title="同步至 B 端" onclick="window._syncFile('${escapeHtml(file.key)}', 'a')">➡️同步</button>`;
              break;
            case 'ONLY_B':
              statusBadge = '<span class="status-badge status-ONLY_B">🟣 仅 B 端</span>';
              syncActionBtn = `<button class="action-btn" title="同步至 A 端" onclick="window._syncFile('${escapeHtml(file.key)}', 'b')">⬅️同步</button>`;
              break;
            case 'DIFF':
              statusBadge = '<span class="status-badge status-DIFF">🟡 内容差异</span>';
              syncActionBtn = `<button class="action-btn diff-sync-btn" title="内容存在差异，点击选择同步方向" onclick="window._openDiffSyncModal('${escapeHtml(file.key)}')">🔄同步</button>`;
              break;
          }

          card.innerHTML = `
            ${thumbHtml}
            <div class="file-details">
              <span class="file-name" title="${escapeHtml(file.name)}">${escapeHtml(file.name)}</span>
              <div class="file-meta">
                <span>${formatSize(file.size)}</span>
                ${statusBadge}
              </div>
              <div class="file-time-row">
                <span class="file-time">${formatTime(file.last_modified)}</span>
                ${syncActionBtn}
              </div>
              <div class="file-card-actions">
                ${file.is_image ? `<button class="action-btn preview-btn" title="大图预览">👁️</button>` : ''}
                <button class="action-btn copy-url-btn" title="复制外网访问链接 (优先使用CDN)">🔗</button>
                <button class="action-btn copy-btn" title="复制文件 Key 路径">📋</button>
                <button class="action-btn acl-btn" title="设置读写权限 (ACL)">🛡️</button>
                <button class="action-btn delete delete-btn" title="彻底删除 (双端同步)">🗑️</button>
              </div>
            </div>
          `;

          // 绑定事件
          if (file.is_image) {
            const thumbEl = card.querySelector('.file-thumb');
            const prevBtn = card.querySelector('.preview-btn');
            const openPrev = () => openImagePreview(file.name, file.key);
            if (thumbEl) thumbEl.addEventListener('click', openPrev);
            if (prevBtn) prevBtn.addEventListener('click', openPrev);
          }

          card.querySelector('.copy-url-btn').addEventListener('click', () => copyFileUrl(file));
          card.querySelector('.copy-btn').addEventListener('click', () => {
            navigator.clipboard.writeText(file.key).then(() => {
              showToast(`已复制路径: ${file.key}`, 'info');
            });
          });
          card.querySelector('.acl-btn').addEventListener('click', () => openAclModal(file.key, file.name));
          card.querySelector('.delete-btn').addEventListener('click', () => {
            confirmDeleteFile(file.key, file.name);
          });

          els.filesGrid.appendChild(card);
        });
      } else {
        els.filesSection.style.display = 'none';
        els.filesGrid.innerHTML = '';
        els.filesTableBody.innerHTML = '';
      }

    } else {
      // ==================== 列表表格模式 (LIST VIEW) ====================
      // 在列表模式下，文件夹与文件统一呈现在整齐的表格中
      els.foldersSection.style.display = 'none';
      els.foldersGrid.innerHTML = '';

      els.filesSection.style.display = 'flex';
      els.filesGrid.style.display = 'none';
      els.filesGrid.innerHTML = '';
      els.filesListContainer.style.display = 'block';
      els.filesTableBody.innerHTML = '';

      // 动态更新板块标题
      if (els.filesSectionLabel) {
        if (subDirs.length > 0 && files.length > 0) {
          els.filesSectionLabel.textContent = `📋 内容列表 (共 ${subDirs.length} 个目录，${files.length} 个文件)`;
        } else if (subDirs.length > 0) {
          els.filesSectionLabel.textContent = `📂 目录列表 (共 ${subDirs.length} 个目录)`;
        } else {
          els.filesSectionLabel.textContent = `📄 文件列表 (共 ${files.length} 个文件)`;
        }
      }

      // 1. 若同时有目录和文件，增加「目录」分组提示行
      if (subDirs.length > 0 && files.length > 0) {
        const groupHeader = document.createElement('tr');
        groupHeader.className = 'table-group-header-row';
        groupHeader.innerHTML = `
          <td colspan="6">
            <span class="table-group-pill">📂 目录 (FOLDERS · ${subDirs.length})</span>
          </td>
        `;
        els.filesTableBody.appendChild(groupHeader);
      }

      // 渲染文件夹表格行
      if (subDirs.length > 0) {
        subDirs.forEach(sub => {
          const canDelete = !isRoot && !isRootSyncDir(sub.path);
          const tr = document.createElement('tr');
          tr.className = 'folder-table-row';
          tr.title = `点击进入目录: ${sub.name}`;
          tr.innerHTML = `
            <td>
              <span class="file-row-icon folder-table-icon">📁</span>
            </td>
            <td>
              <div class="file-row-name-box">
                <span class="file-row-name folder-row-name" title="${escapeHtml(sub.name)}">${escapeHtml(sub.name)}</span>
                <span class="file-row-key">${escapeHtml(sub.path)}</span>
              </div>
            </td>
            <td>
              <span class="status-badge status-folder">${isRoot ? '📁 白名单根目录' : '📁 子文件夹'}</span>
            </td>
            <td><span class="table-dash">-</span></td>
            <td><span class="table-dash">-</span></td>
            <td style="text-align: right;">
              <div class="file-actions">
                ${canDelete ? `
                  <button type="button" class="action-btn delete-btn" title="彻底删除此文件夹 (双端同步)">🗑️ 删除</button>
                ` : `
                  <span class="action-hint-text">默认同步根目录</span>
                `}
              </div>
            </td>
          `;

          // 点击整行进入该目录
          tr.addEventListener('click', () => {
            loadDirectory(sub.path);
          });

          // 点击删除按钮阻止冒泡
          if (canDelete) {
            const delBtn = tr.querySelector('.delete-btn');
            if (delBtn) {
              delBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                confirmDeleteDirectory(sub.path, sub.name);
              });
            }
          }

          els.filesTableBody.appendChild(tr);
        });
      }

      // 2. 若同时有目录和文件，增加「文件」分组提示行
      if (subDirs.length > 0 && files.length > 0) {
        const groupHeader = document.createElement('tr');
        groupHeader.className = 'table-group-header-row';
        groupHeader.innerHTML = `
          <td colspan="6">
            <span class="table-group-pill">📄 文件 (FILES · ${files.length})</span>
          </td>
        `;
        els.filesTableBody.appendChild(groupHeader);
      }

      // 渲染文件表格行
      if (files.length > 0) {
        files.forEach(file => {
          const tr = document.createElement('tr');

          // 缩略图单元格
          let thumbTd = '';
          if (file.is_image) {
            thumbTd = `
              <td>
                <img class="file-row-thumb" src="/api/preview?key=${encodeURIComponent(file.key)}" alt="${escapeHtml(file.name)}" loading="lazy">
              </td>
            `;
          } else {
            thumbTd = `
              <td>
                <span class="file-row-icon">📄</span>
              </td>
            `;
          }

          // 状态徽章
          let statusBadge = '';
          let syncActionBtn = '';
          switch (file.status) {
            case 'SYNCED':
              statusBadge = '<span class="status-badge status-SYNCED">🟢 双端一致</span>';
              break;
            case 'ONLY_A':
              statusBadge = '<span class="status-badge status-ONLY_A">🔵 仅 A 端</span>';
              syncActionBtn = `<button class="action-btn" title="同步至 B 端" onclick="window._syncFile('${escapeHtml(file.key)}', 'a')">➡️同步</button>`;
              break;
            case 'ONLY_B':
              statusBadge = '<span class="status-badge status-ONLY_B">🟣 仅 B 端</span>';
              syncActionBtn = `<button class="action-btn" title="同步至 A 端" onclick="window._syncFile('${escapeHtml(file.key)}', 'b')">⬅️同步</button>`;
              break;
            case 'DIFF':
              statusBadge = '<span class="status-badge status-DIFF">🟡 内容差异</span>';
              syncActionBtn = `<button class="action-btn diff-sync-btn" title="内容存在差异，点击选择同步方向" onclick="window._openDiffSyncModal('${escapeHtml(file.key)}')">🔄同步</button>`;
              break;
          }

          tr.innerHTML = `
            ${thumbTd}
            <td>
              <div class="file-row-name-box">
                <span class="file-row-name" title="${escapeHtml(file.name)}">${escapeHtml(file.name)}</span>
                <span class="file-row-key">${escapeHtml(file.key)}</span>
              </div>
            </td>
            <td>${statusBadge}</td>
            <td>${formatSize(file.size)}</td>
            <td>${formatTime(file.last_modified)}</td>
            <td style="text-align: right;">
              <div class="file-actions">
                ${syncActionBtn}
                ${file.is_image ? `<button class="action-btn preview-btn" title="大图预览">👁️</button>` : ''}
                <button class="action-btn copy-url-btn" title="复制外网访问链接 (优先使用CDN)">🔗地址</button>
                <button class="action-btn copy-btn" title="复制文件 Key 路径">📋复制</button>
                <button class="action-btn acl-btn" title="设置读写权限 (ACL)">🔒ACL</button>
                <button class="action-btn delete-btn" title="彻底删除 (双端同步)">🗑️删除</button>
              </div>
            </td>
          `;

          // 绑定行内事件
          if (file.is_image) {
            const thumbEl = tr.querySelector('.file-row-thumb');
            const prevBtn = tr.querySelector('.preview-btn');
            const openPrev = () => openImagePreview(file.name, file.key);
            if (thumbEl) thumbEl.addEventListener('click', openPrev);
            if (prevBtn) prevBtn.addEventListener('click', openPrev);
          }

          tr.querySelector('.copy-url-btn').addEventListener('click', () => copyFileUrl(file));
          tr.querySelector('.copy-btn').addEventListener('click', () => {
            navigator.clipboard.writeText(file.key).then(() => {
              showToast(`已复制路径: ${file.key}`, 'info');
            });
          });
          tr.querySelector('.acl-btn').addEventListener('click', () => openAclModal(file.key, file.name));
          tr.querySelector('.delete-btn').addEventListener('click', () => confirmDeleteFile(file.key, file.name));

          els.filesTableBody.appendChild(tr);
        });
      }
    }
  }

  // 渲染面包屑
  function renderBreadcrumbs(crumbs) {
    els.breadcrumbs.innerHTML = '';
    const safeCrumbs = Array.isArray(crumbs) ? crumbs : [];
    safeCrumbs.forEach((c, idx) => {
      const isLast = idx === safeCrumbs.length - 1;
      const span = document.createElement('span');
      span.className = `breadcrumb-item ${isLast ? 'active' : ''}`;
      span.textContent = c.name;
      if (!isLast) {
        span.addEventListener('click', () => {
          loadDirectory(c.path);
        });
      }
      els.breadcrumbs.appendChild(span);

      if (!isLast) {
        const sep = document.createElement('span');
        sep.className = 'breadcrumb-separator';
        sep.textContent = '/';
        els.breadcrumbs.appendChild(sep);
      }
    });
  }

  // 视图模式切换 (网格 / 列表)
  function setViewMode(mode) {
    currentViewMode = mode;
    localStorage.setItem('oss_sync_view_mode', mode);
    if (els.btnViewGrid) els.btnViewGrid.classList.toggle('active', mode === 'grid');
    if (els.btnViewList) els.btnViewList.classList.toggle('active', mode === 'list');
    if (currentListing) {
      renderListing(currentListing);
    }
  }

  // 获取文件在 Bucket A 与 Bucket B 上的所有可用外网访问链接（支持分别展示 OSS 原生与 CDN 加速域名）
  function getAllFileUrls(file) {
    const cleanKey = (file.key || '').replace(/^[/\\]+/, '');
    const a = (systemStatus && systemStatus.bucket_a) || {};
    const b = (systemStatus && systemStatus.bucket_b) || {};

    const cleanDomain = (d) => (d || '').trim().replace(/^https?:\/\//, '').replace(/\/+$/, '');

    const formatOSS = (bucket, endpoint) => {
      const bName = (bucket || '').trim();
      const ep = cleanDomain(endpoint);
      return (bName && ep) ? `https://${bName}.${ep}/${cleanKey}` : '';
    };

    const formatCDN = (cdn) => {
      const d = cleanDomain(cdn);
      return d ? `https://${d}/${cleanKey}` : '';
    };

    return {
      bucketA: {
        bucketName: (a.bucket_name || '').trim() || 'Bucket A',
        inBucket: file.in_a !== false,
        cdnUrl: formatCDN(a.cdn_domain),
        ossUrl: formatOSS(a.bucket_name, a.endpoint)
      },
      bucketB: {
        bucketName: (b.bucket_name || '').trim() || 'Bucket B',
        inBucket: file.in_b !== false,
        cdnUrl: formatCDN(b.cdn_domain),
        ossUrl: formatOSS(b.bucket_name, b.endpoint)
      }
    };
  }

  // 复制文件外网访问链接：
  // 规则：支持生成带有效期的只读鉴权签名链接（私有文件可免密访问）与公共链接，支持自由选择 Bucket A 与 Bucket B
  let currentCopyUrlFile = null;

  function copyFileUrl(file) {
    currentCopyUrlFile = file;
    openCopyUrlModal(file);
  }

  // 执行写入剪贴板并提示
  function doCopyUrl(url, label) {
    navigator.clipboard.writeText(url).then(() => {
      showToast(`已复制 ${label} 链接`, 'success');
    }).catch(() => {
      showToast(`已复制 ${label} 链接`, 'info');
    });
  }

  // 打开获取地址弹窗
  function openCopyUrlModal(file) {
    if (!els.modalCopyUrl) return;

    if (els.copyUrlFileName) {
      els.copyUrlFileName.textContent = file.name || file.key;
      els.copyUrlFileName.title = file.key;
    }

    // 默认恢复 3600 秒和签名开启
    if (els.copyUrlExpiresInput && (!els.copyUrlExpiresInput.value || parseInt(els.copyUrlExpiresInput.value, 10) <= 0)) {
      els.copyUrlExpiresInput.value = '3600';
    }
    updatePresetPillState();

    els.modalCopyUrl.classList.add('show');
    generateAndRenderUrls();
  }

  function closeCopyUrlModal() {
    if (els.modalCopyUrl) {
      els.modalCopyUrl.classList.remove('show');
    }
  }

  function updatePresetPillState() {
    const val = els.copyUrlExpiresInput ? parseInt(els.copyUrlExpiresInput.value, 10) : 3600;
    const container = els.modalCopyUrl;
    if (container) {
      container.querySelectorAll('.preset-pill').forEach(pill => {
        const sec = parseInt(pill.getAttribute('data-sec'), 10);
        pill.classList.toggle('active', sec === val);
      });
    }
  }

  // 请求后台生成并渲染链接
  async function generateAndRenderUrls() {
    if (!currentCopyUrlFile) return;

    const container = document.getElementById('copy-url-options-list');
    if (!container) return;

    let expires = els.copyUrlExpiresInput ? parseInt(els.copyUrlExpiresInput.value, 10) : 3600;
    if (isNaN(expires) || expires <= 0) expires = 3600;
    if (expires > 604800) expires = 604800; // 阿里云限制最长 7 天

    const signRadio = document.querySelector('input[name="copy-url-sign-option"]:checked');
    const isSigned = signRadio ? signRadio.value === '1' : true;

    container.innerHTML = `<div class="copy-url-loading">⏳ 正在向阿里云 OSS 生成${isSigned ? '带鉴权签名的' : ''}访问地址...</div>`;

    try {
      const res = await fetch(`/api/file-urls?key=${encodeURIComponent(currentCopyUrlFile.key)}&expires=${expires}&signed=${isSigned ? '1' : '0'}`);
      const data = await res.json();
      if (data.success && data.data) {
        renderUrlOptionsList(data.data, isSigned, expires);
      } else {
        checkIfTokenExpired(data.message);
        // 回退本地未签名计算
        const fallback = getAllFileUrls(currentCopyUrlFile);
        renderUrlOptionsFallback(fallback, isSigned, expires);
        if (data.message) showToast('获取签名地址提示: ' + data.message, 'warning');
      }
    } catch (e) {
      checkIfTokenExpired(e.message);
      const fallback = getAllFileUrls(currentCopyUrlFile);
      renderUrlOptionsFallback(fallback, isSigned, expires);
    }
  }

  function renderUrlOptionsList(urlData, isSigned, expires) {
    const container = document.getElementById('copy-url-options-list');
    if (!container) return;

    const file = currentCopyUrlFile || {};
    const inA = file.in_a !== false;
    const inB = file.in_b !== false;

    let html = '';

    // 1. Bucket A (源端)
    if (urlData.bucket_a && (urlData.bucket_a.cdn_url || urlData.bucket_a.oss_url)) {
      const bA = urlData.bucket_a;
      html += `
        <div class="copy-bucket-group">
          <div class="copy-bucket-group-header">
            <span class="copy-url-bucket-icon">📦</span>
            <span class="copy-url-bucket-label">Bucket A (源端)</span>
            <span class="copy-url-bucket-name">${escapeHtml(bA.bucket_name || 'A')}</span>
            ${!inA ? '<span class="copy-url-sync-tag">⚠️ 当前未同步至该端</span>' : ''}
          </div>
          <div class="copy-url-rows">
            ${bA.oss_url ? `
              <div class="copy-url-row" data-url="${escapeHtml(bA.oss_url)}" data-label="Bucket A (${isSigned && bA.is_signed ? 'OSS 授权签名' : '原生 OSS'})" title="点击复制 Bucket A 的访问链接">
                ${isSigned && bA.is_signed ? 
                  `<span class="copy-url-tag tag-signed">🔐 OSS 授权签名 (${expires}秒有效)</span>` : 
                  `<span class="copy-url-tag tag-oss">☁️ OSS 原生</span>`
                }
                <span class="copy-url-preview-text">${escapeHtml(bA.oss_url)}</span>
                <button type="button" class="btn ${isSigned && bA.is_signed ? 'btn-primary' : 'btn-secondary'} btn-sm btn-copy-row">📋 复制</button>
              </div>
            ` : ''}
            ${bA.cdn_url ? `
              <div class="copy-url-row" data-url="${escapeHtml(bA.cdn_url)}" data-label="Bucket A (CDN 加速)" title="点击复制 Bucket A 的 CDN 加速链接">
                <span class="copy-url-tag tag-cdn">🚀 CDN 加速</span>
                <span class="copy-url-preview-text">${escapeHtml(bA.cdn_url)}</span>
                <button type="button" class="btn btn-primary btn-sm btn-copy-row">📋 复制</button>
              </div>
            ` : ''}
          </div>
        </div>
      `;
    }

    // 2. Bucket B (目标端)
    if (urlData.bucket_b && (urlData.bucket_b.cdn_url || urlData.bucket_b.oss_url)) {
      const bB = urlData.bucket_b;
      html += `
        <div class="copy-bucket-group">
          <div class="copy-bucket-group-header">
            <span class="copy-url-bucket-icon">🎯</span>
            <span class="copy-url-bucket-label">Bucket B (目标端)</span>
            <span class="copy-url-bucket-name">${escapeHtml(bB.bucket_name || 'B')}</span>
            ${!inB ? '<span class="copy-url-sync-tag">⚠️ 当前未同步至该端</span>' : ''}
          </div>
          <div class="copy-url-rows">
            ${bB.oss_url ? `
              <div class="copy-url-row" data-url="${escapeHtml(bB.oss_url)}" data-label="Bucket B (${isSigned && bB.is_signed ? 'OSS 授权签名' : '原生 OSS'})" title="点击复制 Bucket B 的访问链接">
                ${isSigned && bB.is_signed ? 
                  `<span class="copy-url-tag tag-signed">🔐 OSS 授权签名 (${expires}秒有效)</span>` : 
                  `<span class="copy-url-tag tag-oss">☁️ OSS 原生</span>`
                }
                <span class="copy-url-preview-text">${escapeHtml(bB.oss_url)}</span>
                <button type="button" class="btn ${isSigned && bB.is_signed ? 'btn-primary' : 'btn-secondary'} btn-sm btn-copy-row">📋 复制</button>
              </div>
            ` : ''}
            ${bB.cdn_url ? `
              <div class="copy-url-row" data-url="${escapeHtml(bB.cdn_url)}" data-label="Bucket B (CDN 加速)" title="点击复制 Bucket B 的 CDN 加速链接">
                <span class="copy-url-tag tag-cdn">🚀 CDN 加速</span>
                <span class="copy-url-preview-text">${escapeHtml(bB.cdn_url)}</span>
                <button type="button" class="btn btn-primary btn-sm btn-copy-row">📋 复制</button>
              </div>
            ` : ''}
          </div>
        </div>
      `;
    }

    if (!html) {
      html = '<div class="copy-url-loading" style="color: var(--warning);">⚠️ 未能生成访问链接，请检查存储桶配置</div>';
    }

    container.innerHTML = html;

    // 绑定点击复制交互
    bindCopyRowEvents(container);
  }

  function renderUrlOptionsFallback(fallback, isSigned, expires) {
    const fakeData = {
      bucket_a: {
        bucket_name: fallback.bucketA.bucketName,
        oss_url: fallback.bucketA.ossUrl,
        cdn_url: fallback.bucketA.cdnUrl,
        is_signed: false
      },
      bucket_b: {
        bucket_name: fallback.bucketB.bucketName,
        oss_url: fallback.bucketB.ossUrl,
        cdn_url: fallback.bucketB.cdnUrl,
        is_signed: false
      }
    };
    renderUrlOptionsList(fakeData, false, expires);
  }

  function bindCopyRowEvents(container) {
    container.querySelectorAll('.copy-url-row').forEach(row => {
      const url = row.getAttribute('data-url');
      const label = row.getAttribute('data-label');
      const btn = row.querySelector('.btn-copy-row');

      const triggerCopy = (e) => {
        if (e) e.stopPropagation();
        doCopyUrl(url, label);
        if (btn) {
          btn.textContent = '✓ 已复制!';
          btn.classList.add('copied');
        }
        setTimeout(() => {
          closeCopyUrlModal();
        }, 400);
      };

      row.onclick = triggerCopy;
      if (btn) btn.onclick = triggerCopy;
    });
  }

  // 计算文件通用外网访问链接（用于向后兼容）
  function getFilePublicUrl(file) {
    if (file && file.url) return file.url;
    const urls = getAllFileUrls(file);
    if (urls.bucketB.cdnUrl) return urls.bucketB.cdnUrl;
    if (urls.bucketB.ossUrl) return urls.bucketB.ossUrl;
    if (urls.bucketA.cdnUrl) return urls.bucketA.cdnUrl;
    if (urls.bucketA.ossUrl) return urls.bucketA.ossUrl;
    return (file.key || '').replace(/^[/\\]+/, '');
  }

  // 读写权限 (ACL) 弹窗与设置
  async function openAclModal(key, name) {
    currentAclTarget = { key, name, acl: 'default' };
    if (els.aclFileName) els.aclFileName.textContent = name;
    updateAclPills('default');
    if (els.modalAcl) els.modalAcl.classList.add('show');

    try {
      const res = await fetch(`/api/file-acl?key=${encodeURIComponent(key)}`);
      const data = await res.json();
      if (data.success && data.data && data.data.acl) {
        currentAclTarget.acl = data.data.acl.toLowerCase();
        updateAclPills(currentAclTarget.acl);
      }
    } catch (e) {
      console.warn('获取 ACL 状态异常:', e);
    }
  }

  function updateAclPills(val) {
    if (!els.aclBtnGroup) return;
    els.aclBtnGroup.querySelectorAll('.acl-pill').forEach(btn => {
      btn.classList.toggle('active', btn.getAttribute('data-val') === val);
    });
    if (els.aclDesc) {
      els.aclDesc.innerHTML = (aclDescriptions[val] || aclDescriptions['default']).replace(/\s+/, '&nbsp;&nbsp;');
    }
  }

  async function saveFileAcl() {
    if (!currentAclTarget.key) return;
    try {
      els.btnConfirmAcl.disabled = true;
      els.btnConfirmAcl.textContent = '授权中...';
      const res = await fetch('/api/file-acl', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          key: currentAclTarget.key,
          acl: currentAclTarget.acl
        })
      });
      const data = await res.json();
      if (data.success) {
        showToast(data.message || '权限已生效 (双端同步完成)', 'success');
        els.modalAcl.classList.remove('show');
      } else {
        checkIfTokenExpired(data.message);
        showToast(data.message || '授权失败', 'error');
      }
    } catch (e) {
      checkIfTokenExpired(e.message);
      showToast('设置权限异常: ' + e.message, 'error');
    } finally {
      els.btnConfirmAcl.disabled = false;
      els.btnConfirmAcl.textContent = '确认授权';
    }
  }

  // 4. 拖拽上传与文件夹递归解析实现
  function setupDragAndDrop() {
    window.addEventListener('dragenter', (e) => {
      e.preventDefault();
      dragCounter++;
      if (dragCounter === 1) {
        els.dropTargetName.textContent = currentDir ? `当前目标目录: [ ${currentDir} ]` : '根目录';
        els.dropOverlay.classList.add('active');
      }
    });

    window.addEventListener('dragover', (e) => {
      e.preventDefault();
    });

    window.addEventListener('dragleave', (e) => {
      e.preventDefault();
      dragCounter--;
      if (dragCounter <= 0) {
        dragCounter = 0;
        els.dropOverlay.classList.remove('active');
      }
    });

    window.addEventListener('drop', async (e) => {
      e.preventDefault();
      dragCounter = 0;
      els.dropOverlay.classList.remove('active');

      const items = e.dataTransfer && e.dataTransfer.items;
      if (items && items.length > 0) {
        try {
          const fileEntries = await scanDroppedItems(items);
          if (fileEntries.length > 0) {
            uploadFileEntries(fileEntries);
            return;
          }
        } catch (err) {
          console.warn('扫描拖拽项异常，降级普通文件处理:', err);
        }
      }

      if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        const fileEntries = Array.from(e.dataTransfer.files).map(f => ({ file: f, relPath: f.name }));
        uploadFileEntries(fileEntries);
      }
    });

    // 按钮选择文件上传
    els.btnUpload.addEventListener('click', () => {
      els.fileInput.click();
    });

    els.fileInput.addEventListener('change', (e) => {
      if (e.target.files && e.target.files.length > 0) {
        const fileEntries = Array.from(e.target.files).map(f => ({ file: f, relPath: f.name }));
        uploadFileEntries(fileEntries);
        e.target.value = '';
      }
    });

    // 按钮选择文件夹上传
    if (els.btnUploadFolder && els.folderInput) {
      els.btnUploadFolder.addEventListener('click', () => {
        els.folderInput.click();
      });

      els.folderInput.addEventListener('change', (e) => {
        if (e.target.files && e.target.files.length > 0) {
          const fileEntries = Array.from(e.target.files).map(f => ({
            file: f,
            relPath: f.webkitRelativePath || f.name
          }));
          uploadFileEntries(fileEntries);
          e.target.value = '';
        }
      });
    }
  }

  // 递归扫描拖拽放入的所有文件及子文件夹
  async function scanDroppedItems(items) {
    const results = [];
    const queue = [];

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.kind === 'file') {
        if (typeof item.webkitGetAsEntry === 'function') {
          const entry = item.webkitGetAsEntry();
          if (entry) {
            queue.push({ entry, path: '' });
            continue;
          }
        }
        const f = item.getAsFile();
        if (f) results.push({ file: f, relPath: f.name });
      }
    }

    while (queue.length > 0) {
      const { entry, path } = queue.shift();
      if (entry.isFile) {
        const f = await new Promise((res) => entry.file(res, () => res(null)));
        if (f) {
          results.push({ file: f, relPath: path ? `${path}/${f.name}` : f.name });
        }
      } else if (entry.isDirectory) {
        const dirReader = entry.createReader();
        const dirPath = path ? `${path}/${entry.name}` : entry.name;
        const subEntries = await new Promise((res) => {
          let all = [];
          function readNext() {
            dirReader.readEntries((entries) => {
              if (!entries || !entries.length) {
                res(all);
              } else {
                all = all.concat(Array.from(entries));
                readNext();
              }
            }, () => res(all));
          }
          readNext();
        });

        if (subEntries.length === 0) {
          // 空文件夹，在 OSS 中保留一个以 / 结尾的目录占位对象
          results.push({ file: new Blob([]), relPath: `${dirPath}/`, isDir: true });
        } else {
          for (const child of subEntries) {
            queue.push({ entry: child, path: dirPath });
          }
        }
      }
    }
    return results;
  }

  // 统一分批执行文件/文件夹上传
  async function uploadFileEntries(fileEntries) {
    if (!currentDir) {
      showToast('请先点击进入具体的同步目录（如 images/），再进行上传', 'warning');
      return;
    }
    if (!fileEntries || fileEntries.length === 0) {
      showToast('未选择任何有效文件', 'warning');
      return;
    }

    const total = fileEntries.length;
    showToast(`正在双端并发上传并覆盖 ${total} 个文件 (保留完整文件夹层级)...`, 'info');

    // 分批次上传（每批最多 30 个文件，杜绝单次 payload 过大导致 Failed to fetch）
    const BATCH_SIZE = 30;
    let uploadedTotal = 0;
    let failCount = 0;

    for (let i = 0; i < total; i += BATCH_SIZE) {
      const batch = fileEntries.slice(i, i + BATCH_SIZE);
      const formData = new FormData();
      formData.append('dir', currentDir);
      for (const item of batch) {
        formData.append('files', item.file, item.relPath);
      }

      try {
        const res = await fetch('/api/upload', {
          method: 'POST',
          body: formData
        });
        const data = await res.json();
        if (data.success) {
          uploadedTotal += batch.length;
        } else {
          failCount += batch.length;
          checkIfTokenExpired(data.message);
          showToast(data.message || '部分文件上传失败', 'error');
        }
      } catch (err) {
        failCount += batch.length;
        checkIfTokenExpired(err.message);
        showToast('上传批次网络异常: ' + err.message, 'error');
      }
    }

    if (failCount === 0) {
      showToast(`✅ 全部 ${uploadedTotal} 个文件及子文件夹已成功双端上传覆盖！`, 'success');
    } else {
      showToast(`上传结束: 成功 ${uploadedTotal} 个，失败 ${failCount} 个`, 'warning');
    }

    loadDirectory(currentDir);
  }

  // 兼容调用
  async function handleUploadFiles(files) {
    const fileEntries = Array.from(files).map(f => ({ file: f, relPath: f.name }));
    return uploadFileEntries(fileEntries);
  }

  // 5. 彻底删除文件/文件夹交互 (应用内置原生对话框，告别浏览器自带弹窗)
  let currentTargetDeleteType = 'file'; // 'file' | 'dir'
  let currentTargetDeleteKey = '';
  let currentTargetDeleteName = '';

  function confirmDeleteFile(key, filename) {
    currentTargetDeleteType = 'file';
    currentTargetDeleteKey = key;
    currentTargetDeleteName = filename || key;

    if (els.deleteModalTitle) els.deleteModalTitle.textContent = '🗑️ 彻底删除文件';
    if (els.deleteWarningText) {
      els.deleteWarningText.innerHTML = '<strong>危险操作提醒：</strong> 此操作将同时从 <strong>Bucket A (源端)</strong> 与 <strong>Bucket B (目标端)</strong> 中双端彻底删除该文件，<strong>操作不可逆且无法找回</strong>！';
    }
    if (els.deleteInfoTypeLabel) els.deleteInfoTypeLabel.textContent = '目标文件:';
    if (els.deleteTargetName) els.deleteTargetName.textContent = filename || key;
    if (els.deleteTargetKey) els.deleteTargetKey.textContent = key;

    if (els.btnConfirmDeleteAction) {
      els.btnConfirmDeleteAction.disabled = false;
      els.btnConfirmDeleteAction.innerHTML = '🗑️ 彻底删除';
    }

    if (els.modalConfirmDelete) {
      els.modalConfirmDelete.classList.add('show');
    }
  }

  function confirmDeleteDirectory(dirPath, dirName) {
    currentTargetDeleteType = 'dir';
    currentTargetDeleteKey = dirPath;
    currentTargetDeleteName = dirName || dirPath;

    if (els.deleteModalTitle) els.deleteModalTitle.textContent = '🗑️ 彻底删除文件夹';
    if (els.deleteWarningText) {
      els.deleteWarningText.innerHTML = '<strong>危险操作提醒：</strong> 此操作将同时从 <strong>Bucket A (源端)</strong> 与 <strong>Bucket B (目标端)</strong> 中双端彻底递归删除该文件夹及其内部<strong>所有文件与子目录</strong>，<strong>操作不可逆且无法找回</strong>！';
    }
    if (els.deleteInfoTypeLabel) els.deleteInfoTypeLabel.textContent = '目标文件夹:';
    if (els.deleteTargetName) els.deleteTargetName.textContent = dirName || dirPath;
    if (els.deleteTargetKey) els.deleteTargetKey.textContent = dirPath;

    if (els.btnConfirmDeleteAction) {
      els.btnConfirmDeleteAction.disabled = false;
      els.btnConfirmDeleteAction.innerHTML = '🗑️ 彻底删除文件夹';
    }

    if (els.modalConfirmDelete) {
      els.modalConfirmDelete.classList.add('show');
    }
  }

  function closeDeleteModal() {
    if (els.modalConfirmDelete) {
      els.modalConfirmDelete.classList.remove('show');
    }
    currentTargetDeleteType = 'file';
    currentTargetDeleteKey = '';
    currentTargetDeleteName = '';
  }

  async function executeDeleteAction() {
    if (!currentTargetDeleteKey) return;

    const isDir = currentTargetDeleteType === 'dir';

    if (els.btnConfirmDeleteAction) {
      els.btnConfirmDeleteAction.disabled = true;
      els.btnConfirmDeleteAction.textContent = isDir ? '正在清理删除文件夹...' : '正在删除...';
    }

    showToast(isDir ? '正在双端彻底递归清理删除文件夹...' : '正在双端同步彻底删除文件...', 'info');
    try {
      const url = isDir ? '/api/delete-dir' : '/api/delete';
      const payload = isDir ? { dir: currentTargetDeleteKey } : { key: currentTargetDeleteKey };

      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (data.success) {
        showToast(data.message || (isDir ? '文件夹已双端彻底删除！' : '文件已成功双端删除！'), 'success');
        const deletedKey = currentTargetDeleteKey;
        closeDeleteModal();

        if (isDir) {
          // 如果删除的是当前打开的目录（或当前目录位于被删目录内），导航回其父级目录
          const cleanCur = (currentDir || '').replace(/^[/\\]+|[/\\]+$/g, '');
          const cleanDel = (deletedKey || '').replace(/^[/\\]+|[/\\]+$/g, '');
          if (cleanCur === cleanDel || cleanCur.startsWith(cleanDel + '/')) {
            const parts = cleanDel.split('/').filter(Boolean);
            parts.pop();
            const parentDir = parts.length > 0 ? parts.join('/') + '/' : '';
            loadDirectory(parentDir);
            return;
          }
        }
        loadDirectory(currentDir);
      } else {
        checkIfTokenExpired(data.message);
        showToast('删除失败: ' + data.message, 'error');
      }
    } catch (e) {
      checkIfTokenExpired(e.message);
      showToast('删除请求异常: ' + e.message, 'error');
    } finally {
      if (els.btnConfirmDeleteAction) {
        els.btnConfirmDeleteAction.disabled = false;
        els.btnConfirmDeleteAction.innerHTML = isDir ? '🗑️ 彻底删除文件夹' : '🗑️ 彻底删除';
      }
    }
  }

  // 6. 新建子目录交互 (应用内置原生对话框，告别浏览器自带弹窗)
  function promptNewDir() {
    if (!currentDir) {
      showToast('请先进入具体的同步根目录后再创建子目录', 'warning');
      return;
    }

    if (els.newDirParentPath) {
      els.newDirParentPath.textContent = `📂 ${currentDir}`;
    }
    if (els.inputNewDirName) {
      els.inputNewDirName.value = '';
    }
    if (els.btnConfirmNewDir) {
      els.btnConfirmNewDir.disabled = false;
      els.btnConfirmNewDir.innerHTML = '📁 确定创建';
    }

    if (els.modalNewDir) {
      els.modalNewDir.classList.add('show');
      setTimeout(() => {
        if (els.inputNewDirName) els.inputNewDirName.focus();
      }, 60);
    }
  }

  function closeNewDirModal() {
    if (els.modalNewDir) {
      els.modalNewDir.classList.remove('show');
    }
  }

  async function submitNewDir() {
    if (!currentDir) return;
    const name = els.inputNewDirName ? els.inputNewDirName.value.trim() : '';
    if (!name) {
      showToast('请输入子目录名称', 'warning');
      if (els.inputNewDirName) els.inputNewDirName.focus();
      return;
    }

    if (els.btnConfirmNewDir) {
      els.btnConfirmNewDir.disabled = true;
      els.btnConfirmNewDir.textContent = '创建中...';
    }

    try {
      const res = await fetch('/api/create-dir', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          parent_dir: currentDir,
          name: name
        })
      });
      const data = await res.json();
      if (data.success) {
        showToast(data.message || '目录创建成功！', 'success');
        closeNewDirModal();
        loadDirectory(currentDir);
      } else {
        checkIfTokenExpired(data.message);
        showToast(data.message || '创建目录失败', 'error');
      }
    } catch (e) {
      checkIfTokenExpired(e.message);
      showToast('创建目录异常: ' + e.message, 'error');
    } finally {
      if (els.btnConfirmNewDir) {
        els.btnConfirmNewDir.disabled = false;
        els.btnConfirmNewDir.innerHTML = '📁 确定创建';
      }
    }
  }

  // 7. 单文件同步
  window._syncFile = function (key, from) {
    fetch('/api/sync-file', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key, from })
    })
    .then(res => res.json())
    .then(data => {
      if (data.success) {
        showToast(data.message, 'success');
        loadDirectory(currentDir);
      } else {
        checkIfTokenExpired(data.message);
        showToast(data.message, 'error');
      }
    })
    .catch(e => {
      checkIfTokenExpired(e.message);
      showToast('对齐同步失败: ' + e.message, 'error');
    });
  };

  // 7.1 差异解决弹窗与双向对齐同步
  let currentDiffTargetKey = '';

  function openDiffSyncModal(key) {
    currentDiffTargetKey = key;
    const file = currentListing && currentListing.files ? currentListing.files.find(f => f.key === key) : null;
    if (!file) {
      showToast('未找到指定文件数据', 'error');
      return;
    }

    const aName = (systemStatus && systemStatus.bucket_a && systemStatus.bucket_a.bucket_name) || '存储桶 A';
    const bName = (systemStatus && systemStatus.bucket_b && systemStatus.bucket_b.bucket_name) || '存储桶 B';

    if (els.diffSyncFileName) {
      els.diffSyncFileName.textContent = file.name || file.key;
      els.diffSyncFileName.title = file.key;
    }

    // Bucket A
    if (els.diffBnameA) els.diffBnameA.textContent = aName;
    if (els.diffSizeA) els.diffSizeA.textContent = formatSize(file.size_a !== undefined ? file.size_a : file.size);
    if (els.diffTimeA) els.diffTimeA.textContent = formatTime(file.mod_time_a || file.last_modified);
    if (els.diffEtagA) {
      const etag = file.etag_a || '-';
      els.diffEtagA.textContent = etag;
      els.diffEtagA.title = etag;
    }

    // Bucket B
    if (els.diffBnameB) els.diffBnameB.textContent = bName;
    if (els.diffSizeB) els.diffSizeB.textContent = formatSize(file.size_b !== undefined ? file.size_b : file.size);
    if (els.diffTimeB) els.diffTimeB.textContent = formatTime(file.mod_time_b || file.last_modified);
    if (els.diffEtagB) {
      const etag = file.etag_b || '-';
      els.diffEtagB.textContent = etag;
      els.diffEtagB.title = etag;
    }

    // 恢复按钮初始状态
    if (els.btnDiffSyncA) {
      els.btnDiffSyncA.disabled = false;
      els.btnDiffSyncA.innerHTML = `➡️ 以 A 为准同步到 B <small>(覆盖 B 端)</small>`;
    }
    if (els.btnDiffSyncB) {
      els.btnDiffSyncB.disabled = false;
      els.btnDiffSyncB.innerHTML = `⬅️ 以 B 为准同步到 A <small>(覆盖 A 端)</small>`;
    }

    if (els.modalDiffSync) {
      els.modalDiffSync.classList.add('show');
    }
  }

  function closeDiffSyncModal() {
    if (els.modalDiffSync) {
      els.modalDiffSync.classList.remove('show');
    }
  }

  async function executeDiffSync(direction) {
    if (!currentDiffTargetKey) return;
    const fromA = direction === 'a';
    const activeBtn = fromA ? els.btnDiffSyncA : els.btnDiffSyncB;
    const otherBtn = fromA ? els.btnDiffSyncB : els.btnDiffSyncA;
    const srcName = fromA ? (els.diffBnameA ? els.diffBnameA.textContent : 'A 端') : (els.diffBnameB ? els.diffBnameB.textContent : 'B 端');
    const dstName = fromA ? (els.diffBnameB ? els.diffBnameB.textContent : 'B 端') : (els.diffBnameA ? els.diffBnameA.textContent : 'A 端');

    if (activeBtn) {
      activeBtn.disabled = true;
      activeBtn.innerHTML = `⏳ 正在执行覆盖同步...`;
    }
    if (otherBtn) otherBtn.disabled = true;

    try {
      const res = await fetch('/api/sync-file', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key: currentDiffTargetKey, from: direction })
      });
      const data = await res.json();
      if (data.success) {
        showToast(`已成功以【${srcName}】为基准覆盖【${dstName}】！`, 'success');
        closeDiffSyncModal();
        loadDirectory(currentDir);
      } else {
        checkIfTokenExpired(data.message);
        showToast(data.message || '同步失败', 'error');
      }
    } catch (e) {
      checkIfTokenExpired(e.message);
      showToast('同步异常: ' + e.message, 'error');
    } finally {
      if (activeBtn) {
        activeBtn.disabled = false;
        if (fromA) {
          activeBtn.innerHTML = `➡️ 以 A 为准同步到 B <small>(覆盖 B 端)</small>`;
        } else {
          activeBtn.innerHTML = `⬅️ 以 B 为准同步到 A <small>(覆盖 A 端)</small>`;
        }
      }
      if (otherBtn) otherBtn.disabled = false;
    }
  }

  window._openDiffSyncModal = openDiffSyncModal;

  // 8. 大图预览
  function openImagePreview(title, key) {
    els.previewTitle.textContent = title;
    els.previewImg.src = `/api/preview?key=${encodeURIComponent(key)}`;
    els.modalPreview.classList.add('show');
  }

  // 9. 配置弹窗交互
  async function openConfigModal() {
    try {
      const res = await fetch('/api/config');
      const data = await res.json();
      if (data.success && data.data) {
        editConfig = data.data;
        renderConfigModal();
        els.modalConfig.classList.add('show');
      }
    } catch (e) {
      showToast('获取配置失败: ' + e.message, 'error');
    }
  }

  // 切换认证模式 Tab
  function switchAuthTab(which, tab) {
    authState[which].authType = tab;
    const tabsContainer = which === 'a' ? els.tabsA : els.tabsB;
    if (tabsContainer) {
      tabsContainer.querySelectorAll('.auth-tab-btn').forEach(btn => {
        btn.classList.toggle('active', btn.getAttribute('data-tab') === tab);
      });
    }

    const isA = which === 'a';
    const paneAk = isA ? els.paneAAk : els.paneBAk;
    const paneAcc = isA ? els.paneAAccount : els.paneBAccount;
    const paneCode = isA ? els.paneACode : els.paneBCode;

    if (paneAk) paneAk.classList.toggle('active', tab === 'ak');
    if (paneAcc) paneAcc.classList.toggle('active', tab === 'account');
    if (paneCode) paneCode.classList.toggle('active', tab === 'code');
  }

  // 发起账号登录（唤起独立的 Edge/Chrome 窗口）
  async function startAccountLogin(which) {
    const isA = which === 'a';
    const target = isA ? 'bucket_a' : 'bucket_b';
    const badge = isA ? els.accBadgeA : els.accBadgeB;
    const msg = isA ? els.accMsgA : els.accMsgB;
    const btn = isA ? els.btnOpenLoginA : els.btnOpenLoginB;

    if (authState[which].pollTimer) {
      clearInterval(authState[which].pollTimer);
      authState[which].pollTimer = null;
    }

    if (badge) {
      badge.className = 'status-badge status-DIFF';
      badge.textContent = '⏳ 正在唤起登录...';
    }
    if (msg) msg.textContent = '正在拉起 Edge/Chrome 独立窗口，请在弹窗中完成登录...';
    if (btn) btn.disabled = true;

    try {
      const res = await fetch('/api/auth/open-login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target })
      });
      const data = await res.json();
      if (!data.success) {
        if (badge) badge.textContent = '❌ 唤起失败';
        if (msg) msg.textContent = data.message || '无法唤起浏览器窗口';
        if (btn) btn.disabled = false;
        showToast(data.message || '唤起登录窗口失败', 'error');
        return;
      }

      if (badge) badge.textContent = '⏳ 等待在弹窗中登录...';
      if (msg) msg.textContent = data.message || '请在弹出窗口中完成 RAM 子账号或扫码登录...';
      showToast('已唤起阿里云登录窗口，请在弹出的浏览器中完成登录', 'info');

      // 弹窗唤起后允许再次点击重新拉起
      setTimeout(() => {
        if (btn) btn.disabled = false;
      }, 1500);

      authState[which].pollTimer = setInterval(async () => {
        try {
          const sRes = await fetch(`/api/auth/login-status?target=${target}`);
          const sData = await sRes.json();
          if (sData.success && sData.data) {
            const info = sData.data;
            if (info.status === 'success' && info.credentials) {
              clearInterval(authState[which].pollTimer);
              authState[which].pollTimer = null;
              if (btn) btn.disabled = false;

              if (badge) {
                badge.className = 'status-badge status-SYNCED';
                badge.textContent = '✅ 登录成功';
              }
              if (msg) msg.textContent = '已获取并解密临时访问凭据 (有效期约 6 小时)！';

              authState[which].ak = info.credentials.access_key_id;
              authState[which].sk = info.credentials.access_key_secret;
              authState[which].securityToken = info.credentials.security_token;

              showToast(`${isA ? 'Bucket A' : 'Bucket B'} 账号登录成功！`, 'success');
              hideReloginBanner();

              // 自动持久化新凭证到后端配置并刷新工作区
              const cfgA = getActiveBucketConfig('a');
              const cfgB = getActiveBucketConfig('b');
              if (cfgA.access_key_id && cfgB.access_key_id && cfgA.bucket_name && cfgB.bucket_name) {
                fetch('/api/config', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    bucket_a: cfgA,
                    bucket_b: cfgB,
                    sync_dirs: editConfig.sync_dirs || []
                  })
                }).then(r => r.json()).then(saveRes => {
                  if (saveRes.success) {
                    showToast(`${isA ? 'Bucket A' : 'Bucket B'} 登录凭据已更新并自动生效！`, 'success');
                    checkSystemStatus();
                    loadDirectory(currentDir);
                  }
                }).catch(err => console.error('自动保存凭据异常:', err));
              } else {
                listBuckets(which, true);
              }
            } else if (info.status === 'failed') {
              clearInterval(authState[which].pollTimer);
              authState[which].pollTimer = null;
              if (btn) btn.disabled = false;
              if (badge) {
                badge.className = 'status-badge status-DIFF';
                badge.textContent = '❌ 凭据置换失败';
              }
              if (msg) msg.textContent = info.message;
            } else if (info.status === 'closed') {
              clearInterval(authState[which].pollTimer);
              authState[which].pollTimer = null;
              if (btn) btn.disabled = false;
              if (badge) {
                badge.className = 'status-badge status-DIFF';
                badge.textContent = '⏹️ 登录窗口已关闭';
              }
              if (msg) msg.textContent = '若需重新登录，请直接点击上方按钮重新打开登录窗口。';
            }
          }
        } catch (err) {
          // ignore transient poll error
        }
      }, 1000);

    } catch (e) {
      if (badge) badge.textContent = '❌ 网络请求异常';
      if (msg) msg.textContent = e.message;
      if (btn) btn.disabled = false;
      showToast('唤起登录异常: ' + e.message, 'error');
    }
  }

  // 获取 Bucket 列表
  async function listBuckets(which, silent = false) {
    const isA = which === 'a';
    const cfg = getActiveBucketConfig(which);
    const selectEl = isA ? els.bucketSelectA : els.bucketSelectB;
    const inputEl = isA ? els.bucketAAcc : els.bucketBAcc;

    if (!cfg.access_key_id || !cfg.access_key_secret) {
      if (!silent) showToast('请先完成账号登录以获取访问凭据', 'warning');
      return;
    }

    try {
      const res = await fetch('/api/auth/list-buckets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(cfg)
      });
      const data = await res.json();
      if (data.success && Array.isArray(data.data) && data.data.length > 0) {
        selectEl.innerHTML = '<option value="">-- 点击选择发现的 Bucket --</option>';
        data.data.forEach(b => {
          const opt = document.createElement('option');
          opt.value = b;
          opt.textContent = b;
          if (b === inputEl.value.trim()) opt.selected = true;
          selectEl.appendChild(opt);
        });
        selectEl.style.display = 'block';
        if (!silent) showToast(`成功发现 ${data.data.length} 个可用 Bucket`, 'success');
      } else {
        if (!silent) showToast(data.message || '未获取到 Bucket 列表，可手动输入', 'info');
      }
    } catch (e) {
      if (!silent) showToast('获取 Bucket 异常: ' + e.message, 'error');
    }
  }

  // 解析授权码
  async function parseAuthCode(which) {
    const isA = which === 'a';
    const inputArea = isA ? els.authcodeInputA : els.authcodeInputB;
    const epInput = isA ? els.endpointACode : els.endpointBCode;
    const bkInput = isA ? els.bucketACode : els.bucketBCode;

    const raw = inputArea.value.trim();
    if (!raw) {
      showToast('请先粘贴授权码内容', 'warning');
      return;
    }

    try {
      const res = await fetch('/api/auth/parse-code', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: raw })
      });
      const data = await res.json();
      if (data.success && data.data) {
        const item = data.data;
        authState[which].ak = item.access_key_id;
        authState[which].sk = item.access_key_secret;
        authState[which].securityToken = item.security_token || '';
        authState[which].endpoint = item.endpoint || 'oss-cn-hangzhou.aliyuncs.com';
        authState[which].bucket = item.bucket_name || '';

        epInput.value = authState[which].endpoint;
        bkInput.value = authState[which].bucket;

        if (item.path_prefix) {
          if (!editConfig.sync_dirs.includes(item.path_prefix)) {
            editConfig.sync_dirs.push(item.path_prefix);
            renderSyncDirsChips();
          }
        }

        showToast('授权码解析成功！已自动填入 Endpoint 与 Bucket', 'success');
      } else {
        showToast(data.message || '解析失败，请检查授权码格式', 'error');
      }
    } catch (e) {
      showToast('解析授权码异常: ' + e.message, 'error');
    }
  }

  // 获取当前所选 Tab 的 Bucket 配置
  function getActiveBucketConfig(which) {
    const isA = which === 'a';
    const tab = authState[which].authType;
    let ak = '', sk = '', token = '', endpoint = '', bucket = '';

    if (tab === 'ak') {
      ak = (isA ? els.akA : els.akB).value.trim();
      sk = (isA ? els.skA : els.skB).value.trim();
      endpoint = (isA ? els.endpointA : els.endpointB).value.trim() || 'oss-cn-hangzhou.aliyuncs.com';
      bucket = (isA ? els.bucketA : els.bucketB).value.trim();
    } else if (tab === 'account') {
      ak = authState[which].ak;
      sk = authState[which].sk;
      token = authState[which].securityToken;
      endpoint = (isA ? els.endpointAAcc : els.endpointBAcc).value.trim() || 'oss-cn-hangzhou.aliyuncs.com';
      bucket = (isA ? els.bucketAAcc : els.bucketBAcc).value.trim();
    } else if (tab === 'code') {
      ak = authState[which].ak;
      sk = authState[which].sk;
      token = authState[which].securityToken;
      endpoint = (isA ? els.endpointACode : els.endpointBCode).value.trim() || 'oss-cn-hangzhou.aliyuncs.com';
      bucket = (isA ? els.bucketACode : els.bucketBCode).value.trim();
    }

    const cdnDomain = (isA ? els.cdnDomainA : els.cdnDomainB) ? (isA ? els.cdnDomainA : els.cdnDomainB).value.trim() : '';

    return {
      name: isA ? "源端 (Bucket A)" : "目标端 (Bucket B)",
      access_key_id: ak,
      access_key_secret: sk,
      security_token: token,
      endpoint: endpoint,
      bucket_name: bucket,
      cdn_domain: cdnDomain,
      auth_type: tab
    };
  }

  function renderConfigModal() {
    // 渲染 A 端
    const a = editConfig.bucket_a || {};
    if (els.cdnDomainA) els.cdnDomainA.value = a.cdn_domain || '';
    if (a.auth_type === 'account') {
      authState.a.authType = 'account';
      authState.a.ak = a.access_key_id || '';
      authState.a.sk = a.access_key_secret || '';
      authState.a.securityToken = a.security_token || '';
      els.endpointAAcc.value = a.endpoint || 'oss-cn-hangzhou.aliyuncs.com';
      els.bucketAAcc.value = a.bucket_name || '';
      if (a.security_token) {
        els.accBadgeA.className = 'status-badge status-SYNCED';
        els.accBadgeA.textContent = '✅ 已获取凭证';
        els.accMsgA.textContent = '已有可用临时凭据。若凭证过期或失效，请直接点击上方按钮重新登录刷新。';
        if (els.btnOpenLoginA) els.btnOpenLoginA.textContent = '🔄 重新登录刷新凭据';
      } else {
        if (els.btnOpenLoginA) els.btnOpenLoginA.textContent = '🔗 打开阿里云登录窗口';
      }
      switchAuthTab('a', 'account');
    } else if (a.auth_type === 'auth_code') {
      authState.a.authType = 'code';
      authState.a.ak = a.access_key_id || '';
      authState.a.sk = a.access_key_secret || '';
      authState.a.securityToken = a.security_token || '';
      els.endpointACode.value = a.endpoint || 'oss-cn-hangzhou.aliyuncs.com';
      els.bucketACode.value = a.bucket_name || '';
      switchAuthTab('a', 'code');
    } else {
      authState.a.authType = 'ak';
      els.akA.value = a.access_key_id || '';
      els.skA.value = a.access_key_secret || '';
      els.endpointA.value = a.endpoint || 'oss-cn-hangzhou.aliyuncs.com';
      els.bucketA.value = a.bucket_name || '';
      switchAuthTab('a', 'ak');
    }

    // 渲染 B 端
    const b = editConfig.bucket_b || {};
    if (els.cdnDomainB) els.cdnDomainB.value = b.cdn_domain || '';
    if (b.auth_type === 'account') {
      authState.b.authType = 'account';
      authState.b.ak = b.access_key_id || '';
      authState.b.sk = b.access_key_secret || '';
      authState.b.securityToken = b.security_token || '';
      els.endpointBAcc.value = b.endpoint || 'oss-cn-hangzhou.aliyuncs.com';
      els.bucketBAcc.value = b.bucket_name || '';
      if (b.security_token) {
        els.accBadgeB.className = 'status-badge status-SYNCED';
        els.accBadgeB.textContent = '✅ 已获取凭证';
        els.accMsgB.textContent = '已有可用临时凭据。若凭证过期或失效，请直接点击上方按钮重新登录刷新。';
        if (els.btnOpenLoginB) els.btnOpenLoginB.textContent = '🔄 重新登录刷新凭据';
      } else {
        if (els.btnOpenLoginB) els.btnOpenLoginB.textContent = '🔗 打开阿里云登录窗口';
      }
      switchAuthTab('b', 'account');
    } else if (b.auth_type === 'auth_code') {
      authState.b.authType = 'code';
      authState.b.ak = b.access_key_id || '';
      authState.b.sk = b.access_key_secret || '';
      authState.b.securityToken = b.security_token || '';
      els.endpointBCode.value = b.endpoint || 'oss-cn-hangzhou.aliyuncs.com';
      els.bucketBCode.value = b.bucket_name || '';
      switchAuthTab('b', 'code');
    } else {
      authState.b.authType = 'ak';
      els.akB.value = b.access_key_id || '';
      els.skB.value = b.access_key_secret || '';
      els.endpointB.value = b.endpoint || 'oss-cn-hangzhou.aliyuncs.com';
      els.bucketB.value = b.bucket_name || '';
      switchAuthTab('b', 'ak');
    }

    renderSyncDirsChips();
  }

  function renderSyncDirsChips() {
    els.syncDirsBox.innerHTML = '';
    if (!editConfig.sync_dirs || editConfig.sync_dirs.length === 0) {
      els.syncDirsBox.innerHTML = '<span style="color:var(--text-muted);font-size:12px;">暂无同步目录，请在下方添加（如 images/）</span>';
      return;
    }

    editConfig.sync_dirs.forEach((dir, idx) => {
      const chip = document.createElement('div');
      chip.className = 'dir-chip';
      chip.innerHTML = `
        <span>📂 ${escapeHtml(dir)}</span>
        <span class="dir-chip-remove" data-idx="${idx}">✕</span>
      `;
      els.syncDirsBox.appendChild(chip);
    });

    els.syncDirsBox.querySelectorAll('.dir-chip-remove').forEach(el => {
      el.addEventListener('click', (e) => {
        const i = parseInt(e.target.getAttribute('data-idx'), 10);
        editConfig.sync_dirs.splice(i, 1);
        renderSyncDirsChips();
      });
    });
  }

  function addSyncDir() {
    let val = els.newDirInput.value.trim().replace(/\\/g, '/');
    if (!val) return;
    while (val.startsWith('/')) val = val.substring(1);
    if (!val.endsWith('/')) val += '/';

    if (!editConfig.sync_dirs) editConfig.sync_dirs = [];
    if (editConfig.sync_dirs.includes(val)) {
      showToast('该目录已存在', 'info');
      return;
    }
    editConfig.sync_dirs.push(val);
    els.newDirInput.value = '';
    renderSyncDirsChips();
  }

  async function saveConfigFromModal() {
    const cfgA = getActiveBucketConfig('a');
    const cfgB = getActiveBucketConfig('b');

    const payload = {
      bucket_a: cfgA,
      bucket_b: cfgB,
      sync_dirs: editConfig.sync_dirs || []
    };

    if (!payload.bucket_a.access_key_id || !payload.bucket_a.bucket_name) {
      showToast('请完整配置 Bucket A (源端) 的凭据与 Bucket 名称', 'warning');
      return;
    }

    if (!payload.bucket_b.access_key_id || !payload.bucket_b.bucket_name) {
      showToast('请完整配置 Bucket B (目标端) 的凭据与 Bucket 名称', 'warning');
      return;
    }

    if (payload.sync_dirs.length === 0) {
      showToast('请至少配置一个同步目录白名单', 'warning');
      return;
    }

    try {
      const res = await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const data = await res.json();
      if (data.success) {
        showToast('配置已保存生效！', 'success');
        els.modalConfig.classList.remove('show');
        checkSystemStatus();
      } else {
        showToast('保存失败: ' + data.message, 'error');
      }
    } catch (e) {
      showToast('保存异常: ' + e.message, 'error');
    }
  }

  // 测试连接
  async function testBucketConn(which) {
    const isA = which === 'a';
    const feedbackEl = isA ? els.feedbackA : els.feedbackB;
    const btnEl = isA ? els.btnTestA : els.btnTestB;
    const reqData = getActiveBucketConfig(which);

    if (!reqData.access_key_id || !reqData.bucket_name) {
      feedbackEl.textContent = '❌ 请先完成凭据与 Bucket 填写';
      feedbackEl.style.color = 'var(--danger)';
      return;
    }

    feedbackEl.textContent = '测试连接中...';
    feedbackEl.style.color = 'var(--text-secondary)';
    btnEl.disabled = true;

    try {
      const res = await fetch('/api/test-conn', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(reqData)
      });
      const data = await res.json();
      if (data.success) {
        feedbackEl.textContent = '✅ 连接有效';
        feedbackEl.style.color = 'var(--success)';
      } else {
        feedbackEl.textContent = '❌ ' + (data.message || '连接失败');
        feedbackEl.style.color = 'var(--danger)';
      }
    } catch (e) {
      feedbackEl.textContent = '❌ 网络请求异常';
      feedbackEl.style.color = 'var(--danger)';
    } finally {
      btnEl.disabled = false;
    }
  }

  function escapeHtml(text) {
    if (!text) return '';
    return String(text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  // 绑定基础事件
  function bindEvents() {
    els.btnSettings.addEventListener('click', openConfigModal);
    els.btnCloseConfig.addEventListener('click', () => els.modalConfig.classList.remove('show'));
    els.btnCancelConfig.addEventListener('click', () => els.modalConfig.classList.remove('show'));
    els.btnSaveConfig.addEventListener('click', saveConfigFromModal);

    // Tab 切换事件绑定
    if (els.tabsA) {
      els.tabsA.querySelectorAll('.auth-tab-btn').forEach(btn => {
        btn.addEventListener('click', () => switchAuthTab('a', btn.getAttribute('data-tab')));
      });
    }
    if (els.tabsB) {
      els.tabsB.querySelectorAll('.auth-tab-btn').forEach(btn => {
        btn.addEventListener('click', () => switchAuthTab('b', btn.getAttribute('data-tab')));
      });
    }

    // 账号登录与授权码按钮绑定
    if (els.btnOpenLoginA) els.btnOpenLoginA.addEventListener('click', () => startAccountLogin('a'));
    if (els.btnOpenLoginB) els.btnOpenLoginB.addEventListener('click', () => startAccountLogin('b'));
    if (els.btnListBucketsA) els.btnListBucketsA.addEventListener('click', () => listBuckets('a'));
    if (els.btnListBucketsB) els.btnListBucketsB.addEventListener('click', () => listBuckets('b'));
    if (els.btnParseCodeA) els.btnParseCodeA.addEventListener('click', () => parseAuthCode('a'));
    if (els.btnParseCodeB) els.btnParseCodeB.addEventListener('click', () => parseAuthCode('b'));

    if (els.bucketSelectA) {
      els.bucketSelectA.addEventListener('change', function () {
        if (this.value) {
          els.bucketAAcc.value = this.value;
          authState.a.bucket = this.value;
        }
      });
    }
    if (els.bucketSelectB) {
      els.bucketSelectB.addEventListener('change', function () {
        if (this.value) {
          els.bucketBAcc.value = this.value;
          authState.b.bucket = this.value;
        }
      });
    }

    els.btnAddSyncDir.addEventListener('click', addSyncDir);
    els.newDirInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        addSyncDir();
      }
    });

    els.btnCopyFromA.addEventListener('click', () => {
      const cfgA = getActiveBucketConfig('a');
      const tab = cfgA.auth_type;

      // 保留 B 端原有的 Bucket 名称（用户在 B 端任意 Tab 输入或保存过的 Bucket 名称）
      const existingBucketB = (els.bucketBAcc ? els.bucketBAcc.value.trim() : '') ||
                             (els.bucketB ? els.bucketB.value.trim() : '') ||
                             (els.bucketBCode ? els.bucketBCode.value.trim() : '');

      switchAuthTab('b', tab);
      if (tab === 'ak') {
        els.akB.value = els.akA.value;
        els.skB.value = els.skA.value;
        els.endpointB.value = els.endpointA.value;
        // 仅保留 B 自身输入的 Bucket 名称，绝不覆盖为 A 端的 Bucket 名称
        if (!els.bucketB.value.trim() && existingBucketB) {
          els.bucketB.value = existingBucketB;
        }
      } else if (tab === 'account') {
        authState.b.ak = authState.a.ak;
        authState.b.sk = authState.a.sk;
        authState.b.securityToken = authState.a.securityToken;
        els.endpointBAcc.value = els.endpointAAcc.value;
        // 仅保留 B 自身输入的 Bucket 名称，绝不覆盖为 A 端的 Bucket 名称
        if (!els.bucketBAcc.value.trim() && existingBucketB) {
          els.bucketBAcc.value = existingBucketB;
        }
        els.accBadgeB.className = els.accBadgeA.className;
        els.accBadgeB.textContent = els.accBadgeA.textContent;
        els.accMsgB.textContent = els.accMsgA.textContent;
        if (els.btnOpenLoginB) els.btnOpenLoginB.textContent = '🔄 重新登录刷新凭据';
        listBuckets('b', true);
      } else if (tab === 'code') {
        authState.b.ak = authState.a.ak;
        authState.b.sk = authState.a.sk;
        authState.b.securityToken = authState.a.securityToken;
        els.authcodeInputB.value = els.authcodeInputA.value;
        els.endpointBCode.value = els.endpointACode.value;
        // 仅保留 B 自身输入的 Bucket 名称，绝不覆盖为 A 端的 Bucket 名称
        if (!els.bucketBCode.value.trim() && existingBucketB) {
          els.bucketBCode.value = existingBucketB;
        }
      }
      // 注意：严格遵循用户要求，不复制 A 端的 CDN 加速域名，保留 B 端原有的 CDN 配置
      showToast('已复用 A 端访问凭据 (保留 B 端 Bucket 与 CDN 配置)', 'info');
    });

    els.btnTestA.addEventListener('click', () => testBucketConn('a'));
    els.btnTestB.addEventListener('click', () => testBucketConn('b'));

    els.toggleSkA.addEventListener('click', () => {
      els.skA.type = els.skA.type === 'password' ? 'text' : 'password';
    });
    els.toggleSkB.addEventListener('click', () => {
      els.skB.type = els.skB.type === 'password' ? 'text' : 'password';
    });

    // 视图模式切换
    if (els.btnViewGrid) {
      els.btnViewGrid.addEventListener('click', () => setViewMode('grid'));
    }
    if (els.btnViewList) {
      els.btnViewList.addEventListener('click', () => setViewMode('list'));
    }

    // ACL 读写权限弹窗事件
    if (els.btnCloseAcl) els.btnCloseAcl.addEventListener('click', () => els.modalAcl.classList.remove('show'));
    if (els.btnCancelAcl) els.btnCancelAcl.addEventListener('click', () => els.modalAcl.classList.remove('show'));
    if (els.btnConfirmAcl) els.btnConfirmAcl.addEventListener('click', saveFileAcl);
    if (els.aclBtnGroup) {
      els.aclBtnGroup.querySelectorAll('.acl-pill').forEach(btn => {
        btn.addEventListener('click', () => {
          const aclVal = btn.getAttribute('data-acl');
          if (aclVal) updateAclPills(aclVal);
        });
      });
    }
    if (els.modalAcl) {
      els.modalAcl.addEventListener('click', (e) => {
        if (e.target === els.modalAcl) els.modalAcl.classList.remove('show');
      });
    }

    // 凭据过期重登横幅按钮事件
    if (els.bannerBtnLoginA) els.bannerBtnLoginA.addEventListener('click', () => startAccountLogin('a'));
    if (els.bannerBtnLoginB) els.bannerBtnLoginB.addEventListener('click', () => startAccountLogin('b'));
    if (els.bannerBtnSettings) els.bannerBtnSettings.addEventListener('click', openConfigModal);
    if (els.bannerBtnClose) els.bannerBtnClose.addEventListener('click', hideReloginBanner);

    if (els.btnRefresh) els.btnRefresh.addEventListener('click', () => loadDirectory(currentDir, true));
    if (els.btnCopyCurrDir) {
      els.btnCopyCurrDir.addEventListener('click', () => {
        if (!currentDir) {
          showToast('当前位于根目录，无路径可复制', 'info');
          return;
        }
        navigator.clipboard.writeText(currentDir).then(() => {
          showToast(`已复制当前目录路径: ${currentDir}`, 'success');
          const origHtml = els.btnCopyCurrDir.innerHTML;
          els.btnCopyCurrDir.innerHTML = '✅ 已复制';
          setTimeout(() => {
            if (els.btnCopyCurrDir) els.btnCopyCurrDir.innerHTML = origHtml;
          }, 1500);
        }).catch(err => {
          showToast('复制失败: ' + err.message, 'error');
        });
      });
    }
    if (els.btnRefreshDir) els.btnRefreshDir.addEventListener('click', () => loadDirectory(currentDir, true));
    els.btnNewDir.addEventListener('click', promptNewDir);
    els.searchInput.addEventListener('input', () => {
      if (currentListing) renderListing(currentListing);
    });

    // 视图切换按钮事件绑定 (网格 / 列表)
    if (els.btnViewGrid) els.btnViewGrid.addEventListener('click', () => setViewMode('grid'));
    if (els.btnViewList) els.btnViewList.addEventListener('click', () => setViewMode('list'));

    // 预览弹窗关闭
    els.btnClosePreview.addEventListener('click', () => {
      els.modalPreview.classList.remove('show');
      els.previewImg.src = '';
    });
    els.modalPreview.addEventListener('click', (e) => {
      if (e.target === els.modalPreview) {
        els.modalPreview.classList.remove('show');
        els.previewImg.src = '';
      }
    });

    // 复制外网链接弹窗关闭
    if (els.btnCloseCopyUrl) els.btnCloseCopyUrl.addEventListener('click', closeCopyUrlModal);
    if (els.btnCancelCopyUrl) els.btnCancelCopyUrl.addEventListener('click', closeCopyUrlModal);
    if (els.modalCopyUrl) {
      els.modalCopyUrl.addEventListener('click', (e) => {
        if (e.target === els.modalCopyUrl) closeCopyUrlModal();
      });
    }

    // 获取地址弹窗参数事件绑定 (有效期、签名是/否、预设快捷标签)
    if (els.btnGenerateSignedUrl) {
      els.btnGenerateSignedUrl.addEventListener('click', generateAndRenderUrls);
    }
    if (els.copyUrlExpiresInput) {
      els.copyUrlExpiresInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') generateAndRenderUrls();
      });
      els.copyUrlExpiresInput.addEventListener('change', updatePresetPillState);
    }
    if (els.modalCopyUrl) {
      els.modalCopyUrl.querySelectorAll('.preset-pill').forEach(pill => {
        pill.addEventListener('click', () => {
          const sec = pill.getAttribute('data-sec');
          if (els.copyUrlExpiresInput) {
            els.copyUrlExpiresInput.value = sec;
            updatePresetPillState();
            generateAndRenderUrls();
          }
        });
      });
      els.modalCopyUrl.querySelectorAll('input[name="copy-url-sign-option"]').forEach(radio => {
        radio.addEventListener('change', generateAndRenderUrls);
      });
    }

    // 差异解决弹窗关闭与按钮事件
    if (els.btnCloseDiffSync) els.btnCloseDiffSync.addEventListener('click', closeDiffSyncModal);
    if (els.btnCancelDiffSync) els.btnCancelDiffSync.addEventListener('click', closeDiffSyncModal);
    if (els.modalDiffSync) {
      els.modalDiffSync.addEventListener('click', (e) => {
        if (e.target === els.modalDiffSync) closeDiffSyncModal();
      });
    }
    if (els.btnDiffSyncA) els.btnDiffSyncA.addEventListener('click', () => executeDiffSync('a'));
    if (els.btnDiffSyncB) els.btnDiffSyncB.addEventListener('click', () => executeDiffSync('b'));

    // 新建目录弹窗事件
    if (els.btnCloseNewDir) els.btnCloseNewDir.addEventListener('click', closeNewDirModal);
    if (els.btnCancelNewDir) els.btnCancelNewDir.addEventListener('click', closeNewDirModal);
    if (els.btnConfirmNewDir) els.btnConfirmNewDir.addEventListener('click', submitNewDir);
    if (els.inputNewDirName) {
      els.inputNewDirName.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') submitNewDir();
      });
    }
    if (els.modalNewDir) {
      els.modalNewDir.addEventListener('click', (e) => {
        if (e.target === els.modalNewDir) closeNewDirModal();
      });
    }

    // 彻底删除文件弹窗事件
    if (els.btnCloseDeleteModal) els.btnCloseDeleteModal.addEventListener('click', closeDeleteModal);
    if (els.btnCancelDeleteModal) els.btnCancelDeleteModal.addEventListener('click', closeDeleteModal);
    if (els.btnConfirmDeleteAction) els.btnConfirmDeleteAction.addEventListener('click', executeDeleteAction);
    if (els.modalConfirmDelete) {
      els.modalConfirmDelete.addEventListener('click', (e) => {
        if (e.target === els.modalConfirmDelete) closeDeleteModal();
      });
    }
    if (els.btnDeleteCurrDir) {
      els.btnDeleteCurrDir.addEventListener('click', () => {
        if (!currentDir || isRootSyncDir(currentDir)) {
          showToast('根目录或白名单同步目录不支持直接删除', 'warning');
          return;
        }
        const clean = currentDir.replace(/^[/\\]+|[/\\]+$/g, '');
        const dirName = clean.split('/').pop() || clean;
        confirmDeleteDirectory(currentDir, dirName);
      });
    }

    // 全局快捷键：ESC 关闭弹窗；F5 / Ctrl+R 就地刷新当前目录（避免网页重载跳回根目录）
    window.addEventListener('keydown', (e) => {
      if (e.key === 'F5' || (e.ctrlKey && (e.key === 'r' || e.key === 'R'))) {
        e.preventDefault();
        loadDirectory(currentDir, true);
        return;
      }
      if (e.key === 'Escape') {
        els.modalConfig.classList.remove('show');
        els.modalPreview.classList.remove('show');
        if (els.modalAcl) els.modalAcl.classList.remove('show');
        if (els.modalCopyUrl) closeCopyUrlModal();
        if (els.modalDiffSync) closeDiffSyncModal();
        if (els.modalNewDir) closeNewDirModal();
        if (els.modalConfirmDelete) closeDeleteModal();
      }
    });

    // 窗口切回/获得焦点时自动静默检查当前目录变动（例如在 OSS 控制台修改完文件后切回软件）
    let lastFocusRefresh = 0;
    window.addEventListener('focus', () => {
      const now = Date.now();
      if (now - lastFocusRefresh > 2000 && currentDir !== undefined) {
        lastFocusRefresh = now;
        loadDirectory(currentDir, false);
      }
    });

    // 长时间停留页面时，定时静默巡检当前目录变动（每 15 秒一次）
    setInterval(() => {
      if (!document.hidden && currentDir !== undefined && systemStatus && systemStatus.configured) {
        loadDirectory(currentDir, false);
      }
    }, 15000);
  }

  // 入口
  function init() {
    setViewMode(currentViewMode);
    bindEvents();
    setupDragAndDrop();
    checkSystemStatus();
  }

  document.addEventListener('DOMContentLoaded', init);
})();
