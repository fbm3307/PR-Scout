import axios from 'axios';
import type { Digest, ListResponse, PRWithReview, ReviewComment, ScanRun } from '../types';

const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

export async function triggerScan(): Promise<ScanRun> {
  const { data } = await apiClient.post<ScanRun>('/scan');
  return data;
}

export async function fetchPRs(params?: Record<string, string | number>): Promise<ListResponse<PRWithReview>> {
  const { data } = await apiClient.get<ListResponse<PRWithReview>>('/prs', { params });
  return data;
}

export async function fetchPR(id: number): Promise<{ pr: PRWithReview; comments: ReviewComment[] }> {
  const { data } = await apiClient.get<{ pr: PRWithReview; comments: ReviewComment[] }>(`/prs/${id}`);
  return data;
}

export async function fetchMyReviews(): Promise<ListResponse<PRWithReview>> {
  const { data } = await apiClient.get<ListResponse<PRWithReview>>('/my-reviews');
  return data;
}

export async function fetchDigest(): Promise<Digest> {
  const { data } = await apiClient.get<Digest>('/digest');
  return data;
}
