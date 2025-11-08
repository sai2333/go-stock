import {createApp} from 'vue'
import naive from 'naive-ui'
import App from './App.vue'
import router from './router/router'
// 引入组件库的少量全局样式变量
import 'tdesign-vue-next/es/style/index.css';

// 预览环境兼容：当未在Wails中运行时，提供必要的 runtime 与 go API stub
if (typeof window !== 'undefined') {
  // runtime stub: 避免 wailsjs/runtime 在 Vite 预览报错
  if (!window.runtime) {
    window.runtime = {
      EventsOnMultiple: () => () => {},
      EventsOff: () => {},
      EventsEmit: () => {},
      Quit: () => {},
      WindowFullscreen: () => {},
      WindowHide: () => {},
      WindowUnfullscreen: () => {},
      WindowSetTitle: () => {},
      Environment: () => Promise.resolve({ platform: 'windows' })
    }
  }
  // go API stub: 前端演示所需最小集合
  if (!window.go) {
    window.go = {
      main: {
        App: {
          GetConfig: () => Promise.resolve({
            openAiEnable: false,
            enableDanmu: false,
            darkTheme: true,
            enableFund: true,
            enableAgent: false,
          }),
          GetVersionInfo: () => Promise.resolve({
            icon: 'https://raw.githubusercontent.com/ArvinLovegood/go-stock/master/build/appicon.png'
          }),
          OpenURL: (url) => { try { window.open(url, '_blank') } catch(e) {} },
          GetFollowedFund: () => Promise.resolve([
            {
              code: '159915',
              name: '易方达创业板ETF',
              netEstimatedUnit: 2.01,
              netEstimatedRate: 0.35,
              netEstimatedUnitTime: '估算 14:30',
              netUnitValue: 2.00,
              netUnitValueDate: '2025-11-06',
              fundBasic: {
                code: '159915',
                name: '易方达创业板ETF',
                type: '指数ETF',
                scale: '500',
                company: '易方达基金',
                manager: '张三',
                rating: '五星',
                trackingTarget: '创业板指',
                netGrowth1: 3.5,
                netGrowth3: 6.2,
                netGrowth6: 12.8,
                netGrowth12: 18.4,
                netGrowth36: -5.1,
                netGrowth60: 22.3,
                netGrowthYTD: 9.7,
                netGrowthAll: 150.2,
              }
            }
          ]),
          GetfundList: (key) => Promise.resolve([
            { code: '159915', name: '易方达创业板ETF' },
            { code: '510630', name: '华夏中证银行ETF' },
            { code: '016533', name: '中庚价值领航' },
          ]),
          FollowFund: (code) => Promise.resolve('关注成功'),
          UnFollowFund: (code) => Promise.resolve('取消关注成功'),
        }
      }
    }
  }
}

const app = createApp(App)
app.use(router)
app.use(naive)
app.mount('#app')