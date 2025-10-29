import { createRouter, createWebHistory } from 'vue-router'
import Indexer from '../pages/rag/indexer.vue'
import Retriever from '../pages/rag/retriever.vue'
import Chat from '../pages/rag/chat.vue'

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/',
            redirect: '/indexer'
        },
        {
            path: '/indexer',
            name: 'indexer',
            component: Indexer
        },
        {
            path: '/retriever',
            name: 'retriever',
            component: Retriever
        },
        {
            path: '/chat',
            name: 'chat',
            component: Chat
        }
    ]
})

export default router