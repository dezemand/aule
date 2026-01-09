import { create } from "zustand";

export type Subscription = {
  topic: string;
  query: any;
};

export type SubscriptionsState = {
  subscriptions: Record<string, Subscription>;

  addSubscription: (id: string, topic: string, query: any) => void;
  removeSubscription: (id: string) => void;
};

export const useSubscriptionsStore = create<SubscriptionsState>((set) => ({
  subscriptions: {},

  addSubscription: (id: string, topic: string, query: any) => {
    set((state) => ({
      subscriptions: {
        ...state.subscriptions,
        [id]: { topic, query },
      },
    }));
  },

  removeSubscription: (id: string) => {
    set((state) => {
      const newSubscriptions = { ...state.subscriptions };
      delete newSubscriptions[id];
      return { subscriptions: newSubscriptions };
    });
  },
}));
