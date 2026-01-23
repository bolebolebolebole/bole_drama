<template>
  <div class="system-settings">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>系统语言</span>
        </div>
      </template>
      
      <el-form label-width="120px">
        <el-form-item label="当前语言">
          <el-radio-group v-model="currentLanguage" disabled>
            <el-radio label="zh">简体中文</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item>
          <el-alert
            title="语言设置"
            type="info"
            :closable="false"
            show-icon
          >
            <template #default>
              <p>系统已固定使用简体中文，无需切换语言。</p>
            </template>
          </el-alert>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const currentLanguage = ref<'zh'>('zh')
</script>

<style scoped>
.system-settings {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

:deep(.el-alert ul) {
  margin-top: 10px;
  padding-left: 20px;
}

:deep(.el-alert li) {
  margin: 5px 0;
}
</style>
