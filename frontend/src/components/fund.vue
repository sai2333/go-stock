<script setup>
import {computed, h, onBeforeMount, onBeforeUnmount, onMounted, reactive, ref} from "vue";
import {Add, ChatboxOutline} from "@vicons/ionicons5";
import {NButton, NEllipsis, NText, useMessage, NSelect, NInputNumber, NSpace, NRadioGroup, NRadio, NTag, NRate} from "naive-ui";
import {
  FollowFund,
  GetConfig,
  GetFollowedFund,
  GetfundList,
  GetVersionInfo, OpenURL,
  UnFollowFund
} from "../../wailsjs/go/main/App";
import {Environment} from "../../wailsjs/runtime";
import vueDanmaku from 'vue3-danmaku'

const danmus = ref([])
const ws = ref(null)
const icon = ref(null)
const message = useMessage()
const modalShow = ref(false)
const data = reactive({
  modelName:"",
  chatId: "",
  question:"",
  name: "",
  code: "",
  fenshiURL:"",
  kURL:"",
  fullscreen: false,
  airesult: "",
  openAiEnable: false,
  loading: true,
  enableDanmu: false,
})

const followList=ref([])
const options=ref([])
const ticker=ref({})

// 指标筛选与排序
const filters = reactive({
  type: '',
  minScale: 0,
  minRating: 0,
  sortKey: 'm1', // m1:近一月, m12:近一年, est:估算涨跌
  sortOrder: 'desc',
})

const typeOptions = [
  { label: '全部类型', value: '' },
  { label: 'ETF/指数', value: '指数' },
  { label: '股票型', value: '股票' },
  { label: '混合型', value: '混合' },
  { label: '债券型', value: '债券' },
]

const ratingOptions = [
  { label: '不限评级', value: 0 },
  { label: '≥ 3星', value: 3 },
  { label: '≥ 4星', value: 4 },
  { label: '≥ 5星', value: 5 },
]

const sortOptions = [
  { label: '近一月收益', value: 'm1' },
  { label: '近一年收益', value: 'm12' },
  { label: '估算涨跌', value: 'est' },
]

function parseScale(scaleStr) {
  if (!scaleStr) return 0
  const s = String(scaleStr).replace(/[^0-9.]/g, '')
  const num = parseFloat(s)
  return isNaN(num) ? 0 : num
}
function parseRatingStars(ratingStr) {
  if (!ratingStr) return 0
  if (typeof ratingStr === 'number') return ratingStr
  // 如 "五星","四星","三星" 或 "5" 等
  if (/五星/.test(ratingStr)) return 5
  if (/四星/.test(ratingStr)) return 4
  if (/三星/.test(ratingStr)) return 3
  const n = parseInt(String(ratingStr).replace(/[^0-9]/g, ''))
  return isNaN(n) ? 0 : n
}

function getMetric(info, key) {
  const fb = info?.fundBasic || {}
  switch (key) {
    case 'm1': return fb.netGrowth1 ?? 0
    case 'm12': return fb.netGrowth12 ?? 0
    case 'est': return info?.netEstimatedRate ?? 0
    default: return 0
  }
}

const displayList = computed(() => {
  let list = Array.isArray(followList.value) ? [...followList.value] : []
  // 类型过滤（包含匹配）
  if (filters.type) {
    list = list.filter(it => String(it?.fundBasic?.type || '').includes(filters.type))
  }
  // 规模过滤（亿元）
  if (filters.minScale && filters.minScale > 0) {
    list = list.filter(it => parseScale(it?.fundBasic?.scale) >= filters.minScale)
  }
  // 评级过滤（星数）
  if (filters.minRating && filters.minRating > 0) {
    list = list.filter(it => parseRatingStars(it?.fundBasic?.rating) >= filters.minRating)
  }
  // 排序
  list.sort((a, b) => {
    const av = Number(getMetric(a, filters.sortKey) || 0)
    const bv = Number(getMetric(b, filters.sortKey) || 0)
    return filters.sortOrder === 'asc' ? av - bv : bv - av
  })
  return list
})

onBeforeMount(()=>{
  GetConfig().then(result => {
    if (result.openAiEnable) {
      data.openAiEnable = true
    }
    if (result.enableDanmu) {
      data.enableDanmu = true
    }
  })
  GetFollowedFund().then(result => {
    followList.value = result
    //console.log("followList",followList.value)
  })
})

onMounted(() => {
  GetVersionInfo().then((res) => {
    icon.value = res.icon;
  });
  // 创建 WebSocket 连接
  ws.value = new WebSocket('ws://8.134.249.145:16688/ws'); // 替换为你的 WebSocket 服务器地址
  //ws.value = new WebSocket('ws://localhost:16688/ws'); // 替换为你的 WebSocket 服务器地址

  ws.value.onopen = () => {
    //console.log('WebSocket 连接已打开');
  };

  ws.value.onmessage = (event) => {
    if(data.enableDanmu){
      danmus.value.push(event.data);
    }
  };

  ws.value.onerror = (error) => {
    console.error('WebSocket 错误:', error);
  };

  ws.value.onclose = () => {
    //console.log('WebSocket 连接已关闭');
  };

  ticker.value=setInterval(() => {
    GetFollowedFund().then(result => {
      followList.value = result
      //console.log("followList",followList.value)
    })
  }, 1000*60)

})

onBeforeUnmount(() => {
  clearInterval(ticker.value)
  ws.value.close()
  message.destroyAll()
})



function SendDanmu(){
  ws.value.send(data.name)
}
function AddFund(){
  FollowFund(data.code).then(result=>{
    if(result === "关注成功"){
      message.success(result)
      GetFollowedFund().then(result => {
        followList.value = result
        //console.log("followList",followList.value)
      })
    } else {
      message.error(result || "关注失败")
    }
  })
}
function unFollow(code){
  UnFollowFund(code).then(result=>{
    if(result){
      message.success("取消关注成功")
      GetFollowedFund().then(result => {
        followList.value = result
        //console.log("followList",followList.value)
      })
    }
  })
}

function getFundList(value){
  GetfundList(value).then(result=>{
    options.value=[]
    result.forEach(item=>{
      options.value.push({
        label: item.name+" ["+item.code+"]",
        value: item.code,
      })
    })
  })
}
function onSelectFund(value){
  data.code=value
  blinkBorder(value)
}
function formatterTitle(title){
  return () => h(NEllipsis,{
    style: {
      'font-size': '16px',
      'max-width': '180px',
    },
  },{default: () => title,}
  )
}

function search(code,name){
  setTimeout(() => {
    // 兼容非 Wails 预览环境：若 Environment 不可用则兜底为浏览器打开
    const getEnv = typeof Environment === 'function'
        ? Environment
        : (() => Promise.resolve({ platform: 'browser' }))

    getEnv()
      .then(env => {
        const url = "https://fund.eastmoney.com/"+code+".html"
        if (env && (env.platform === 'windows' || env.platform === 'darwin' || env.platform === 'linux')) {
          // 桌面环境：直接用浏览器打开，避免预览环境报错
          window.open(url, "_blank", "noreferrer,width=1000,top=100,left=100,status=no,toolbar=no,location=no,scrollbars=no")
        } else {
          // Wails 或其他：走后端统一打开
          try { OpenURL(url) } catch(e) { window.open(url, "_blank") }
        }
      })
      .catch(() => {
        window.open("https://fund.eastmoney.com/"+code+".html","_blank")
      })

  }, 500)
}

function newchart(code,name){
  modalShow.value=true
  data.name=name
  data.code=code
  data.fenshiURL='https://image.sinajs.cn/newchart/v5/fund/nav/ss/'+code+'.gif'+"?t="+Date.now()
}

function blinkBorder(findId){
  // 获取要滚动到的元素
  const element = document.getElementById(findId);
  if (element) {
    // 滚动到该元素
    element.scrollIntoView({ behavior: 'smooth' });
    const pelement = document.getElementById(findId +'_gi');
    if(pelement){
      // 添加闪烁效果
      pelement.classList.add('blink-border');
      // 3秒后移除闪烁效果
      setTimeout(() => {
        pelement.classList.remove('blink-border');
      }, 1000*5);
    }else{
      console.error(`Element with ID ${findId}_gi not found`);
    }
  }
}
</script>

<template>
  <vue-danmaku v-model:danmus="danmus" useSlot style="height:100px; width:100%;z-index: 9;position:absolute; top: 400px; pointer-events: none;" >
    <template v-slot:dm="{ index, danmu }">
      <n-gradient-text type="info">
        <n-icon :component="ChatboxOutline"/>{{ danmu }}
      </n-gradient-text>
    </template>
  </vue-danmaku>
  <n-flex justify="start" >
    <!-- 指标筛选与排序条 -->
    <n-space v-if="followList.length" style="margin: 8px 4px 12px; width: 100%" :wrap="false">
      <n-select v-model:value="filters.type" :options="typeOptions" size="small" style="width: 140px" placeholder="基金类型" clearable />
      <n-input-number v-model:value="filters.minScale" size="small" style="width: 160px" :precision="0" :min="0" placeholder="规模≥(亿元)" />
      <n-select v-model:value="filters.minRating" :options="ratingOptions" size="small" style="width: 140px" placeholder="评级" />
      <n-select v-model:value="filters.sortKey" :options="sortOptions" size="small" style="width: 140px" placeholder="排序指标" />
      <n-radio-group v-model:value="filters.sortOrder" size="small">
        <n-radio value="desc">降序</n-radio>
        <n-radio value="asc">升序</n-radio>
      </n-radio-group>
    </n-space>
    <n-empty v-if="followList.length === 0" description="暂无关注基金。请在右下角搜索框输入基金名称或代码，选择后点击‘关注’（示例：510630、159915、016533）。" style="margin-top: 80px;" />
    <n-grid v-else :x-gap="8" :cols="3"  :y-gap="8" >
      <n-gi :id="info.code+'_gi'" v-for="info in  displayList" style="margin-left: 2px" >
        <n-card :id="info.code" :title="formatterTitle(info.name)">
          <template #header-extra>
            <n-tag size="small"  :bordered="false" type="info">{{info.code}}</n-tag>&nbsp;
            <n-tag size="small"  :bordered="false" type="success" @click="unFollow(info.code)"> 取消关注</n-tag>
          </template>
          <n-flex>
            <n-text size="small" :type="info.netEstimatedRate>0?'error':'success'" :bordered="false" v-if="info.netEstimatedUnit">
              估算净值：{{info.netEstimatedUnit}}&nbsp;
              {{info.netEstimatedRate}} %&nbsp;&nbsp;&nbsp;
              ({{info.netEstimatedUnitTime}})</n-text>
            <br>
            <n-text size="small" :type="info.netEstimatedRate>0?'error':'success'" :bordered="false" v-if="info.netUnitValue">
              单位净值：{{info.netUnitValue}}&nbsp;&nbsp;&nbsp; ({{info.netUnitValueDate}})</n-text>
          </n-flex>
            <n-flex justify="start" style="margin-top: 10px">
            <n-tag size="small" :type="info.fundBasic.netGrowth1>0?'error':'success'" :bordered="false" v-if="info.fundBasic.netGrowth1">近一月：{{info.fundBasic.netGrowth1}}%</n-tag>
            <n-tag size="small" :type="info.fundBasic.netGrowth3>0?'error':'success'" :bordered="false" v-if="info.fundBasic.netGrowth3">近三月：{{info.fundBasic.netGrowth3}}%</n-tag>
            <n-tag size="small" :type="info.fundBasic.netGrowth6>0?'error':'success'" :bordered="false" v-if="info.fundBasic.netGrowth6">近六月：{{info.fundBasic.netGrowth6}}%</n-tag>
            <n-tag size="small" :type="info.fundBasic.netGrowth12>0?'error':'success'" :bordered="false" v-if="info.fundBasic.netGrowth12">近一年：{{info.fundBasic.netGrowth12}}%</n-tag>
            <n-tag size="small" :type="info.fundBasic.netGrowth36>0?'error':'success'" :bordered="false" v-if="info.fundBasic.netGrowth36">近三年：{{info.fundBasic.netGrowth36}}%</n-tag>
            <n-tag size="small" :type="info.fundBasic.netGrowth60>0?'error':'success'" :bordered="false" v-if="info.fundBasic.netGrowth60">近五年：{{info.fundBasic.netGrowth60}}%</n-tag>
            <n-tag size="small" :type="info.fundBasic.netGrowthYTD>0?'error':'success'" :bordered="false" v-if="info.fundBasic.netGrowthYTD" >今年来：{{info.fundBasic.netGrowthYTD}}%</n-tag>
            <n-tag size="small" :type="info.fundBasic.netGrowthAll>0?'error':'success'" :bordered="false" >成立来：{{info.fundBasic.netGrowthAll}}%</n-tag>
          </n-flex>
          <template #footer>
            <n-flex justify="space-between">
              <n-tag size="small"  :bordered="false" type="warning"> {{info.fundBasic.type}}</n-tag>
              <n-tag size="small"  :bordered="false" type="success" v-if="info.fundBasic.scale">规模：{{info.fundBasic.scale}}</n-tag>
              <n-tag size="small"  :bordered="false" type="info"> {{info.fundBasic.company}}：{{info.fundBasic.manager}}</n-tag>
            </n-flex>
            <n-flex justify="space-between" style="margin-top: 6px">
              <n-tag size="small"  :bordered="false" type="primary" v-if="info.fundBasic.trackingTarget">跟踪标的：{{info.fundBasic.trackingTarget}}</n-tag>
              <n-rate readonly :value="parseRatingStars(info.fundBasic.rating)" allow-half :count="5" size="small" style="margin-left:auto" />
            </n-flex>
          </template>
          <template #action>
            <n-flex justify="end">
              <n-button size="tiny" type="error" @click="newchart(info.code,info.name)"> 走势 </n-button>
              <n-button size="tiny" type="warning" @click="search(info.code,info.name)"> 详情 </n-button>
            </n-flex>
          </template>
        </n-card>
      </n-gi>
    </n-grid>
  </n-flex>

  <n-modal v-model:show="modalShow" :title="data.name" style="width: 400px" :preset="'card'">
    <n-image :src="data.fenshiURL"   />
  </n-modal>

  <div style="position: fixed;bottom: 18px;right:5px;z-index: 10;width: 400px">
    <n-input-group >
      <n-auto-complete  v-model:value="data.name"
                        :input-props="{
                                autocomplete: 'disabled',
                              }"
                        :options="options"
                        placeholder="基金名称/代码/弹幕"
                        clearable @update-value="getFundList" :on-select="onSelectFund"/>
        <n-button   type="primary" @click="AddFund" >
            <n-icon :component="Add"/>
          关注
        </n-button>
        <n-button   type="info" @click="SendDanmu" v-if="data.enableDanmu" >
            <n-icon :component="ChatboxOutline"/>
          发送弹幕
        </n-button>
    </n-input-group>
  </div>
</template>

<style scoped>
/* 添加闪烁效果的CSS类 */
.blink-border {
  animation: blink-border 1s linear infinite;
  border: 4px  solid transparent;
}

@keyframes blink-border {
  0% {
    border-color: red;
  }
  50% {
    border-color: transparent;
  }
  100% {
    border-color: red;
  }
}
</style>