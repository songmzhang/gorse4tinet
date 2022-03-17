// Copyright 2020 gorse Project Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package master

import (
	"github.com/stretchr/testify/assert"
	"github.com/zhenghaoz/gorse/config"
	"github.com/zhenghaoz/gorse/storage/cache"
	"github.com/zhenghaoz/gorse/storage/data"
	"strconv"
	"testing"
	"time"
)

func TestMaster_FindItemNeighborsBruteForce(t *testing.T) {
	// create mock master
	m := newMockMaster(t)
	defer m.Close()
	// create config
	m.GorseConfig = &config.Config{}
	m.GorseConfig.Database.CacheSize = 3
	m.GorseConfig.Master.NumJobs = 4
	// collect similar
	items := []data.Item{
		{"0", false, []string{"*"}, time.Now(), []string{"a", "b", "c", "d"}, ""},
		{"1", false, nil, time.Now(), []string{"b", "c", "d"}, ""},
		{"2", false, []string{"*"}, time.Now(), []string{"b", "c"}, ""},
		{"3", false, []string{"*"}, time.Now(), []string{"c"}, ""},
		{"4", false, []string{"*"}, time.Now(), []string{}, ""},
		{"5", false, nil, time.Now(), []string{}, ""},
		{"6", false, []string{"*"}, time.Now(), []string{}, ""},
		{"7", false, nil, time.Now(), []string{}, ""},
		{"8", false, []string{"*"}, time.Now(), []string{"a", "b", "c", "d", "e"}, ""},
		{"9", false, nil, time.Now(), []string{}, ""},
	}
	feedbacks := make([]data.Feedback, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j <= i; j++ {
			feedbacks = append(feedbacks, data.Feedback{
				FeedbackKey: data.FeedbackKey{
					ItemId:       strconv.Itoa(i),
					UserId:       strconv.Itoa(j),
					FeedbackType: "FeedbackType",
				},
				Timestamp: time.Now(),
			})
		}
	}
	var err error
	err = m.DataClient.BatchInsertItems(items)
	assert.NoError(t, err)
	err = m.DataClient.BatchInsertFeedback(feedbacks, true, true, true)
	assert.NoError(t, err)

	// insert hidden item
	err = m.DataClient.BatchInsertItems([]data.Item{{
		ItemId:   "10",
		Labels:   []string{"a", "b", "c", "d", "e"},
		IsHidden: true,
	}})
	assert.NoError(t, err)
	for i := 0; i <= 10; i++ {
		err = m.DataClient.BatchInsertFeedback([]data.Feedback{{
			FeedbackKey: data.FeedbackKey{UserId: strconv.Itoa(i), ItemId: "10", FeedbackType: "FeedbackType"},
		}}, true, true, true)
		assert.NoError(t, err)
	}

	// load mock dataset
	dataset, _, _, _, err := m.LoadDataFromDatabase(m.DataClient, []string{"FeedbackType"}, nil, 0, 0)
	assert.NoError(t, err)

	// similar items (common users)
	m.GorseConfig.Recommend.ItemNeighborType = config.NeighborTypeRelated
	m.runFindItemNeighborsTask(dataset)
	similar, err := m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 11, m.taskMonitor.Tasks[TaskFindItemNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindItemNeighbors].Status)
	// similar items in category (common users)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "9", "*"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "6", "4"}, cache.RemoveScores(similar))

	// similar items (common labels)
	err = m.CacheClient.SetTime(cache.LastModifyItemTime, "8", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.ItemNeighborType = config.NeighborTypeSimilar
	m.runFindItemNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	assert.Equal(t, 11, m.taskMonitor.Tasks[TaskFindItemNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindItemNeighbors].Status)
	// similar items in category (common labels)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "8", "*"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "2", "3"}, cache.RemoveScores(similar))

	// similar items (auto)
	err = m.CacheClient.SetTime(cache.LastModifyItemTime, "8", time.Now())
	assert.NoError(t, err)
	err = m.CacheClient.SetTime(cache.LastModifyItemTime, "9", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.ItemNeighborType = config.NeighborTypeAuto
	m.runFindItemNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 11, m.taskMonitor.Tasks[TaskFindItemNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindItemNeighbors].Status)
}

func TestMaster_FindItemNeighborsIVF(t *testing.T) {
	// create mock master
	m := newMockMaster(t)
	defer m.Close()
	// create config
	m.GorseConfig = &config.Config{}
	m.GorseConfig.Database.CacheSize = 3
	m.GorseConfig.Master.NumJobs = 4
	m.GorseConfig.Recommend.EnableItemNeighborIndex = true
	m.GorseConfig.Recommend.ItemNeighborIndexRecall = 1
	m.GorseConfig.Recommend.ItemNeighborIndexFitEpoch = 10
	// collect similar
	items := []data.Item{
		{"0", false, []string{"*"}, time.Now(), []string{"a", "b", "c", "d"}, ""},
		{"1", false, nil, time.Now(), []string{"b", "c", "d"}, ""},
		{"2", false, []string{"*"}, time.Now(), []string{"b", "c"}, ""},
		{"3", false, []string{"*"}, time.Now(), []string{"c"}, ""},
		{"4", false, []string{"*"}, time.Now(), []string{}, ""},
		{"5", false, nil, time.Now(), []string{}, ""},
		{"6", false, []string{"*"}, time.Now(), []string{}, ""},
		{"7", false, nil, time.Now(), []string{}, ""},
		{"8", false, []string{"*"}, time.Now(), []string{"a", "b", "c", "d", "e"}, ""},
		{"9", false, nil, time.Now(), []string{}, ""},
	}
	feedbacks := make([]data.Feedback, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j <= i; j++ {
			feedbacks = append(feedbacks, data.Feedback{
				FeedbackKey: data.FeedbackKey{
					ItemId:       strconv.Itoa(i),
					UserId:       strconv.Itoa(j),
					FeedbackType: "FeedbackType",
				},
				Timestamp: time.Now(),
			})
		}
	}
	var err error
	err = m.DataClient.BatchInsertItems(items)
	assert.NoError(t, err)
	err = m.DataClient.BatchInsertFeedback(feedbacks, true, true, true)
	assert.NoError(t, err)

	// insert hidden item
	err = m.DataClient.BatchInsertItems([]data.Item{{
		ItemId:   "10",
		Labels:   []string{"a", "b", "c", "d", "e"},
		IsHidden: true,
	}})
	assert.NoError(t, err)
	for i := 0; i <= 10; i++ {
		err = m.DataClient.BatchInsertFeedback([]data.Feedback{{
			FeedbackKey: data.FeedbackKey{UserId: strconv.Itoa(i), ItemId: "10", FeedbackType: "FeedbackType"},
		}}, true, true, true)
		assert.NoError(t, err)
	}

	// load mock dataset
	dataset, _, _, _, err := m.LoadDataFromDatabase(m.DataClient, []string{"FeedbackType"}, nil, 0, 0)
	assert.NoError(t, err)

	// similar items (common users)
	m.GorseConfig.Recommend.ItemNeighborType = config.NeighborTypeRelated
	m.runFindItemNeighborsTask(dataset)
	similar, err := m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 11, m.taskMonitor.Tasks[TaskFindItemNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindItemNeighbors].Status)
	// similar items in category (common users)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "9", "*"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "6", "4"}, cache.RemoveScores(similar))

	// similar items (common labels)
	err = m.CacheClient.SetTime(cache.LastModifyItemTime, "8", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.ItemNeighborType = config.NeighborTypeSimilar
	m.runFindItemNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	assert.Equal(t, 11, m.taskMonitor.Tasks[TaskFindItemNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindItemNeighbors].Status)
	// similar items in category (common labels)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "8", "*"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "2", "3"}, cache.RemoveScores(similar))

	// similar items (auto)
	err = m.CacheClient.SetTime(cache.LastModifyItemTime, "8", time.Now())
	assert.NoError(t, err)
	err = m.CacheClient.SetTime(cache.LastModifyItemTime, "9", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.ItemNeighborType = config.NeighborTypeAuto
	m.runFindItemNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.ItemNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 11, m.taskMonitor.Tasks[TaskFindItemNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindItemNeighbors].Status)
}

func TestMaster_FindUserNeighborsBruteForce(t *testing.T) {
	// create mock master
	m := newMockMaster(t)
	defer m.Close()
	// create config
	m.GorseConfig = &config.Config{}
	m.GorseConfig.Database.CacheSize = 3
	m.GorseConfig.Master.NumJobs = 4
	// collect similar
	users := []data.User{
		{"0", []string{"a", "b", "c", "d"}, nil, ""},
		{"1", []string{"b", "c", "d"}, nil, ""},
		{"2", []string{"b", "c"}, nil, ""},
		{"3", []string{"c"}, nil, ""},
		{"4", []string{}, nil, ""},
		{"5", []string{}, nil, ""},
		{"6", []string{}, nil, ""},
		{"7", []string{}, nil, ""},
		{"8", []string{"a", "b", "c", "d", "e"}, nil, ""},
		{"9", []string{}, nil, ""},
	}
	feedbacks := make([]data.Feedback, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j <= i; j++ {
			feedbacks = append(feedbacks, data.Feedback{
				FeedbackKey: data.FeedbackKey{
					ItemId:       strconv.Itoa(j),
					UserId:       strconv.Itoa(i),
					FeedbackType: "FeedbackType",
				},
				Timestamp: time.Now(),
			})
		}
	}
	var err error
	err = m.DataClient.BatchInsertUsers(users)
	assert.NoError(t, err)
	err = m.DataClient.BatchInsertFeedback(feedbacks, true, true, true)
	assert.NoError(t, err)
	dataset, _, _, _, err := m.LoadDataFromDatabase(m.DataClient, []string{"FeedbackType"}, nil, 0, 0)
	assert.NoError(t, err)

	// similar items (common users)
	m.GorseConfig.Recommend.UserNeighborType = config.NeighborTypeRelated
	m.runFindUserNeighborsTask(dataset)
	similar, err := m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 10, m.taskMonitor.Tasks[TaskFindUserNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindUserNeighbors].Status)

	// similar items (common labels)
	err = m.CacheClient.SetTime(cache.LastModifyUserTime, "8", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.UserNeighborType = config.NeighborTypeSimilar
	m.runFindUserNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	assert.Equal(t, 10, m.taskMonitor.Tasks[TaskFindUserNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindUserNeighbors].Status)

	// similar items (auto)
	err = m.CacheClient.SetTime(cache.LastModifyUserTime, "8", time.Now())
	assert.NoError(t, err)
	err = m.CacheClient.SetTime(cache.LastModifyUserTime, "9", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.UserNeighborType = config.NeighborTypeAuto
	m.runFindUserNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 10, m.taskMonitor.Tasks[TaskFindUserNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindUserNeighbors].Status)
}

func TestMaster_FindUserNeighborsIVF(t *testing.T) {
	// create mock master
	m := newMockMaster(t)
	defer m.Close()
	// create config
	m.GorseConfig = &config.Config{}
	m.GorseConfig.Database.CacheSize = 3
	m.GorseConfig.Master.NumJobs = 4
	m.GorseConfig.Recommend.EnableUserNeighborIndex = true
	m.GorseConfig.Recommend.UserNeighborIndexRecall = 1
	m.GorseConfig.Recommend.UserNeighborIndexFitEpoch = 10
	// collect similar
	users := []data.User{
		{"0", []string{"a", "b", "c", "d"}, nil, ""},
		{"1", []string{"b", "c", "d"}, nil, ""},
		{"2", []string{"b", "c"}, nil, ""},
		{"3", []string{"c"}, nil, ""},
		{"4", []string{}, nil, ""},
		{"5", []string{}, nil, ""},
		{"6", []string{}, nil, ""},
		{"7", []string{}, nil, ""},
		{"8", []string{"a", "b", "c", "d", "e"}, nil, ""},
		{"9", []string{}, nil, ""},
	}
	feedbacks := make([]data.Feedback, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j <= i; j++ {
			feedbacks = append(feedbacks, data.Feedback{
				FeedbackKey: data.FeedbackKey{
					ItemId:       strconv.Itoa(j),
					UserId:       strconv.Itoa(i),
					FeedbackType: "FeedbackType",
				},
				Timestamp: time.Now(),
			})
		}
	}
	var err error
	err = m.DataClient.BatchInsertUsers(users)
	assert.NoError(t, err)
	err = m.DataClient.BatchInsertFeedback(feedbacks, true, true, true)
	assert.NoError(t, err)
	dataset, _, _, _, err := m.LoadDataFromDatabase(m.DataClient, []string{"FeedbackType"}, nil, 0, 0)
	assert.NoError(t, err)

	// similar items (common users)
	m.GorseConfig.Recommend.UserNeighborType = config.NeighborTypeRelated
	m.runFindUserNeighborsTask(dataset)
	similar, err := m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 10, m.taskMonitor.Tasks[TaskFindUserNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindUserNeighbors].Status)

	// similar items (common labels)
	err = m.CacheClient.SetTime(cache.LastModifyUserTime, "8", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.UserNeighborType = config.NeighborTypeSimilar
	m.runFindUserNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	assert.Equal(t, 10, m.taskMonitor.Tasks[TaskFindUserNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindUserNeighbors].Status)

	// similar items (auto)
	err = m.CacheClient.SetTime(cache.LastModifyUserTime, "8", time.Now())
	assert.NoError(t, err)
	err = m.CacheClient.SetTime(cache.LastModifyUserTime, "9", time.Now())
	assert.NoError(t, err)
	m.GorseConfig.Recommend.UserNeighborType = config.NeighborTypeAuto
	m.runFindUserNeighborsTask(dataset)
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "8"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, cache.RemoveScores(similar))
	similar, err = m.CacheClient.GetSorted(cache.Key(cache.UserNeighbors, "9"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []string{"8", "7", "6"}, cache.RemoveScores(similar))
	assert.Equal(t, 10, m.taskMonitor.Tasks[TaskFindUserNeighbors].Done)
	assert.Equal(t, TaskStatusComplete, m.taskMonitor.Tasks[TaskFindUserNeighbors].Status)
}

func TestMaster_LoadDataFromDatabase(t *testing.T) {
	// create mock master
	m := newMockMaster(t)
	defer m.Close()
	// create config
	m.GorseConfig = &config.Config{}
	m.GorseConfig.Database.CacheSize = 3
	m.GorseConfig.Database.PositiveFeedbackType = []string{"positive"}
	m.GorseConfig.Database.ReadFeedbackTypes = []string{"negative"}

	// insert items
	var items []data.Item
	for i := 0; i < 9; i++ {
		items = append(items, data.Item{
			ItemId:     strconv.Itoa(i),
			Timestamp:  time.Date(2000+i, 1, 1, 1, 1, 0, 0, time.UTC),
			Labels:     []string{strconv.Itoa(i % 3)},
			Categories: []string{strconv.Itoa(i % 3)},
		})
	}
	err := m.DataClient.BatchInsertItems(items)
	assert.NoError(t, err)
	err = m.DataClient.BatchInsertItems([]data.Item{{
		ItemId:    "9",
		Timestamp: time.Date(2020, 1, 1, 1, 1, 0, 0, time.UTC),
		IsHidden:  true,
	}})
	assert.NoError(t, err)

	// insert users
	var users []data.User
	for i := 0; i <= 10; i++ {
		users = append(users, data.User{
			UserId: strconv.Itoa(i),
			Labels: []string{strconv.Itoa(i % 5)},
		})
	}
	err = m.DataClient.BatchInsertUsers(users)
	assert.NoError(t, err)

	// insert feedback
	feedbacks := make([]data.Feedback, 0)
	for i := 0; i < 10; i++ {
		// positive feedback
		// item 0: user 0
		// ...
		// item 9: user 0 ... user 9
		for j := 0; j <= i; j++ {
			feedbacks = append(feedbacks, data.Feedback{
				FeedbackKey: data.FeedbackKey{
					ItemId:       strconv.Itoa(i),
					UserId:       strconv.Itoa(j),
					FeedbackType: "positive",
				},
				Timestamp: time.Now(),
			})
		}
		// negative feedback
		// item 0: user 1 .. user 10
		// ...
		// item 9: user 10
		for j := i + 1; j < 11; j++ {
			feedbacks = append(feedbacks, data.Feedback{
				FeedbackKey: data.FeedbackKey{
					ItemId:       strconv.Itoa(i),
					UserId:       strconv.Itoa(j),
					FeedbackType: "negative",
				},
				Timestamp: time.Now(),
			})
		}
	}
	err = m.DataClient.BatchInsertFeedback(feedbacks, false, false, true)
	assert.NoError(t, err)

	// load dataset
	err = m.runLoadDatasetTask()
	assert.NoError(t, err)
	assert.Equal(t, 11, m.rankingTrainSet.UserCount())
	assert.Equal(t, 10, m.rankingTrainSet.ItemCount())
	assert.Equal(t, 11, m.rankingTestSet.UserCount())
	assert.Equal(t, 10, m.rankingTestSet.ItemCount())
	assert.Equal(t, 55, m.rankingTrainSet.Count()+m.rankingTestSet.Count())
	assert.Equal(t, 11, m.clickTrainSet.UserCount())
	assert.Equal(t, 10, m.clickTrainSet.ItemCount())
	assert.Equal(t, 11, m.clickTestSet.UserCount())
	assert.Equal(t, 10, m.clickTestSet.ItemCount())
	assert.Equal(t, int32(3), m.clickTrainSet.Index.CountItemLabels())
	assert.Equal(t, int32(5), m.clickTrainSet.Index.CountUserLabels())
	assert.Equal(t, int32(3), m.clickTestSet.Index.CountItemLabels())
	assert.Equal(t, int32(5), m.clickTestSet.Index.CountUserLabels())
	assert.Equal(t, 90, m.clickTrainSet.Count()+m.clickTestSet.Count())
	assert.Equal(t, 45, m.clickTrainSet.PositiveCount+m.clickTestSet.PositiveCount)
	assert.Equal(t, 45, m.clickTrainSet.NegativeCount+m.clickTestSet.NegativeCount)

	// check latest items
	latest, err := m.CacheClient.GetSorted(cache.Key(cache.LatestItems, ""), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []cache.Scored{
		{items[8].ItemId, float32(items[8].Timestamp.Unix())},
		{items[7].ItemId, float32(items[7].Timestamp.Unix())},
		{items[6].ItemId, float32(items[6].Timestamp.Unix())},
	}, latest)
	latest, err = m.CacheClient.GetSorted(cache.Key(cache.LatestItems, "2"), 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, []cache.Scored{
		{items[8].ItemId, float32(items[8].Timestamp.Unix())},
		{items[5].ItemId, float32(items[5].Timestamp.Unix())},
		{items[2].ItemId, float32(items[2].Timestamp.Unix())},
	}, latest)

	// check popular items
	popular, err := m.CacheClient.GetSorted(cache.Key(cache.PopularItems, ""), 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, []cache.Scored{
		{Id: items[8].ItemId, Score: 9},
		{Id: items[7].ItemId, Score: 8},
		{Id: items[6].ItemId, Score: 7},
	}, popular)
	popular, err = m.CacheClient.GetSorted(cache.Key(cache.PopularItems, "2"), 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, []cache.Scored{
		{Id: items[8].ItemId, Score: 9},
		{Id: items[5].ItemId, Score: 6},
		{Id: items[2].ItemId, Score: 3},
	}, popular)

	// check categories
	categories, err := m.CacheClient.GetSet(cache.ItemCategories)
	assert.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, categories)
	categories, err = m.CacheClient.GetSet(cache.Key(cache.ItemCategories, "2"))
	assert.NoError(t, err)
	assert.Equal(t, []string{"2"}, categories)

}
