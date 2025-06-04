import Route from '@ember/routing/route';

export default class extends Route {
  async model() {
    const res = await fetch('https://httpbin.org/anything');
    return await res.json();
  }
}
