import Route from '@ember/routing/route';

export default class extends Route {
  async model() {
    const res = await fetch(`https://api.github.com/users/256dpi`);
    return await res.json();
  }
}
