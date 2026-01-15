import HomePage from "./views/HomePage.vue"
import DashboardPage from "./views/DashboardPage.vue";

export default [
   { path: '/', component: HomePage },
   { path: '/dashboard/:code',  DashboardPage },
];