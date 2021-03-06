package commands_test

import (
	"fmt"
	"strconv"

	"github.com/concourse/atc"
	. "github.com/concourse/fly/commands"
	"github.com/concourse/go-concourse/concourse"
	fakes "github.com/concourse/go-concourse/concourse/concoursefakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helper Functions", func() {
	Describe("#GetBuild", func() {
		var client *fakes.FakeClient

		expectedBuildID := "123"
		expectedBuildName := "5"
		expectedJobName := "myjob"
		expectedPipelineName := "mypipeline"
		expectedBuild := atc.Build{
			ID:      123,
			Name:    expectedBuildName,
			Status:  "Great Success",
			JobName: expectedJobName,
			URL:     fmt.Sprintf("/pipelines/%s/jobs/%s/builds/%s", expectedPipelineName, expectedJobName, expectedBuildName),
			APIURL:  fmt.Sprintf("api/v1/builds/%s", expectedBuildID),
		}

		BeforeEach(func() {
			client = new(fakes.FakeClient)
		})

		Context("when passed a build id", func() {
			Context("when build exists", func() {
				BeforeEach(func() {
					client.BuildReturns(expectedBuild, true, nil)
				})

				It("returns the build", func() {
					build, err := GetBuild(client, "", expectedBuildID, "")
					Expect(err).NotTo(HaveOccurred())
					Expect(build).To(Equal(expectedBuild))
					Expect(client.BuildCallCount()).To(Equal(1))
					Expect(client.BuildArgsForCall(0)).To(Equal(expectedBuildID))
				})
			})

			Context("when a build does not exist", func() {
				BeforeEach(func() {
					client.BuildReturns(atc.Build{}, false, nil)
				})

				It("returns an error", func() {
					_, err := GetBuild(client, "", expectedBuildID, "")
					Expect(err).To(MatchError("build not found"))
				})
			})
		})

		Context("when passed a pipeline and job name", func() {
			Context("when job exists", func() {
				Context("when the next build exists", func() {
					BeforeEach(func() {
						job := atc.Job{
							Name:      expectedJobName,
							NextBuild: &expectedBuild,
						}
						client.JobReturns(job, true, nil)
					})

					It("returns the next build for that job", func() {
						build, err := GetBuild(client, expectedJobName, "", expectedPipelineName)
						Expect(err).NotTo(HaveOccurred())
						Expect(build).To(Equal(expectedBuild))
						Expect(client.JobCallCount()).To(Equal(1))
						pipelineName, jobName := client.JobArgsForCall(0)
						Expect(pipelineName).To(Equal(expectedPipelineName))
						Expect(jobName).To(Equal(expectedJobName))
					})
				})

				Context("when the only the finished build exists", func() {
					BeforeEach(func() {
						job := atc.Job{
							Name:          expectedJobName,
							FinishedBuild: &expectedBuild,
						}
						client.JobReturns(job, true, nil)
					})

					It("returns the finished build for that job", func() {
						build, err := GetBuild(client, expectedJobName, "", expectedPipelineName)
						Expect(err).NotTo(HaveOccurred())
						Expect(build).To(Equal(expectedBuild))
						Expect(client.JobCallCount()).To(Equal(1))
						pipelineName, jobName := client.JobArgsForCall(0)
						Expect(pipelineName).To(Equal(expectedPipelineName))
						Expect(jobName).To(Equal(expectedJobName))
					})
				})

				Context("when no builds exist", func() {
					BeforeEach(func() {
						job := atc.Job{
							Name: expectedJobName,
						}
						client.JobReturns(job, true, nil)
					})

					It("returns an error", func() {
						_, err := GetBuild(client, expectedJobName, "", expectedPipelineName)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("when job does not exists", func() {
				BeforeEach(func() {
					client.JobReturns(atc.Job{}, false, nil)
				})

				It("returns an error", func() {
					_, err := GetBuild(client, expectedJobName, "", expectedPipelineName)
					Expect(err).To(MatchError("job not found"))
				})
			})
		})

		Context("when passed pipeline, job, and build names", func() {
			Context("when the build exists", func() {
				BeforeEach(func() {
					client.JobBuildReturns(expectedBuild, true, nil)
				})

				It("returns the build", func() {
					build, err := GetBuild(client, expectedJobName, expectedBuildName, expectedPipelineName)
					Expect(err).NotTo(HaveOccurred())
					Expect(build).To(Equal(expectedBuild))
					Expect(client.JobBuildCallCount()).To(Equal(1))
					pipelineName, jobName, buildName := client.JobBuildArgsForCall(0)
					Expect(pipelineName).To(Equal(expectedPipelineName))
					Expect(buildName).To(Equal(expectedBuildName))
					Expect(jobName).To(Equal(expectedJobName))
				})
			})

			Context("when the build does not exist", func() {
				BeforeEach(func() {
					client.JobBuildReturns(atc.Build{}, false, nil)
				})

				It("returns an error", func() {
					_, err := GetBuild(client, expectedJobName, expectedBuildName, expectedPipelineName)
					Expect(err).To(MatchError("build not found"))
				})
			})
		})

		Context("when nothing is passed", func() {
			var allBuilds [300]atc.Build

			expectedOneOffBuild := atc.Build{
				ID:      150,
				Name:    expectedBuildName,
				Status:  "success",
				JobName: "",
				URL:     fmt.Sprintf("/builds/%s", expectedBuildID),
				APIURL:  fmt.Sprintf("api/v1/builds/%s", expectedBuildID),
			}

			BeforeEach(func() {
				for i := 300 - 1; i >= 0; i-- {
					allBuilds[i] = atc.Build{
						ID:      i,
						Name:    strconv.Itoa(i),
						JobName: "some-job",
						URL:     fmt.Sprintf("/jobs/some-job/builds/%s", i),
						APIURL:  fmt.Sprintf("api/v1/builds/%s", i),
					}
				}

				allBuilds[150] = expectedOneOffBuild

				client.BuildsStub = func(page concourse.Page) ([]atc.Build, concourse.Pagination, error) {
					var builds []atc.Build
					if page.Since != 0 {
						builds = allBuilds[page.Since : page.Since+page.Limit]
					} else {
						builds = allBuilds[0:page.Limit]
					}

					pagination := concourse.Pagination{
						Previous: &concourse.Page{
							Limit: page.Limit,
							Until: builds[0].ID,
						},
						Next: &concourse.Page{
							Limit: page.Limit,
							Since: builds[len(builds)-1].ID,
						},
					}

					return builds, pagination, nil
				}
			})

			It("returns latest one off build", func() {
				build, err := GetBuild(client, "", "", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(build).To(Equal(expectedOneOffBuild))
				Expect(client.BuildsCallCount()).To(Equal(2))
			})
		})
	})
})
